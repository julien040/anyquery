// This file defines a pipeline that can be used to modify a SQL query
// before running it on the server.
//
// It works by chaining multiple middlewares
// For example it is used to rewrite MySQL specific queries to SQLite
//
// Once the pipeline is run, the result is printed to the output file
package controller

import (
	"database/sql"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/briandowns/spinner"
	"github.com/charmbracelet/lipgloss"
	"github.com/elk-language/go-prompt"
	"golang.org/x/term"
)

type QueryData struct {

	// The query to exec before/after the query
	// Useful to create temporary tables
	PreExec, PostExec []string

	// The query to run
	SQLQuery string

	// The arguments to pass to the query
	Args []interface{}

	// The result of the query
	Result *sql.Rows

	// The database to connect to
	DB *sql.DB

	// The message to return to the client if Result is nil
	Message string

	// The status code of the message
	//
	//    - inferior or equal 0 => INFO
	//    - equal 1 => WARNING
	//    - greater than 2 => ERROR
	//
	StatusCode int

	// The configuration that will be passed to the middlewares
	// This is a reference to the configuration of the pipeline
	Config middlewareConfiguration
}

// Known configuration keys
//
//   - dot-command: bool => if support for dot command is enabled
//   - outputMode: string => the output specified in output.go
//   - logEnabled: bool => if namespace logging is enabled
//   - logFile: string => the file to log the namespace logs
//   - maxRows: int => the maximum number of rows to return
//   - onceOutputFile: string => the file descriptor to output the result (if empty, fallback to fileOutput)
//   - outputFile: string => the file to output the result
//   - separatorColumn: string => the separator to use for columns
//   - separatorRow: string => the separator to use for rows
type middlewareConfiguration map[string]interface{}

func (mc middlewareConfiguration) GetString(key string, defaultVal string) string {
	if val, ok := mc[key]; ok {
		if strVal, ok := val.(string); ok {
			return strVal
		}
	}
	return defaultVal
}

func (mc middlewareConfiguration) GetBool(key string, defaultVal bool) bool {
	if val, ok := mc[key]; ok {
		if boolVal, ok := val.(bool); ok {
			return boolVal
		}
	}
	return defaultVal
}

func (mc middlewareConfiguration) GetInt(key string, defaultVal int) int {
	if val, ok := mc[key]; ok {
		if intVal, ok := val.(int); ok {
			return intVal
		}
	}
	return defaultVal
}

func (mc middlewareConfiguration) SetString(key string, value string) {
	mc[key] = value
}

func (mc middlewareConfiguration) SetBool(key string, value bool) {
	mc[key] = value
}

func (mc middlewareConfiguration) SetInt(key string, value int) {
	mc[key] = value
}

// middleware is a function that takes a QueryData and returns a boolean
// whether to continue the pipeline or not
//
// If the middleware fails, it must set the QueryData.Result to nil
// and an appropriate message to QueryData.Message with status code 2
type middleware func(querydata *QueryData) bool

type shell struct {
	// Middlewares is the list of middlewares to run for the pipeline
	Middlewares []middleware

	// The DB to run the query on
	DB *sql.DB

	// The configuration that will be passed to the middlewares
	Config middlewareConfiguration

	// Where to output the result
	OutputFile     string
	OutputFileDesc io.Writer

	// The history of the shell
	History []string
}

func (p *shell) AddMiddleware(m middleware) {
	p.Middlewares = append(p.Middlewares, m)
}

// Run runs the pipeline on the query,
// prints the result to the output file or stdout,
// and returns a boolean indicating if the shell must exit
//
// If a middleware returns false, the pipeline stops.
// The pipeline will always run the middlewares in the order they were added
func (p *shell) Run(rawQuery string, args ...interface{}) bool {
	queries := splitMultipleQuery(rawQuery)

	if len(queries) == 0 {
		return false
	}

	// Init the map
	if p.Config == nil {
		p.Config = make(middlewareConfiguration, len(queries))
	}

	for i, query := range queries {
		queryData := QueryData{
			SQLQuery: query,
			Config:   p.Config,
			DB:       p.DB,
			Args:     args,
		}

		// If the query is .read, we read the file
		// and run it recursively
		if strings.HasPrefix(query, ".read") {
			pathToRead := strings.TrimSpace(strings.TrimPrefix(query, ".read"))
			file, err := os.ReadFile(pathToRead)
			if err != nil {
				queryData = QueryData{
					Message:    fmt.Sprintf("Error reading file %s: %s", pathToRead, err.Error()),
					StatusCode: 2,
				}
			} else {
				fileContent := string(file)
				// We run the file content recursively
				mustStop := p.Run(fileContent)
				if mustStop {
					return true
				}
			}
		}
		// If the query is .exit, .quit or \q, we stop the pipeline
		if query == ".exit" || query == ".quit" || query == "\\q" {
			return true
		}

		s := spinner.New(spinner.CharSets[11], 50*time.Millisecond)
		s.Prefix = "Running query... "

		// If the output is a terminal, we start the spinner
		if term.IsTerminal(int(os.Stdout.Fd())) {
			s.Start()
		}

		// For each middleware, run it
		// and check if it returns false
		// if it does, we stop the pipeline
		for _, middleware := range p.Middlewares {
			if !middleware(&queryData) {
				break
			}
		}
		s.Stop()

		var tempOutput io.Writer = p.OutputFileDesc
		/* tempOutputMustClose := false */

		// If the output file is specified, we write the result to it
		// and save it for later execution in p.OutputFileDesc
		// We also set it to tempOutput to write the result to it
		confPath := p.Config.GetString("outputFile", "")
		doNotModifyOutput := p.Config.GetBool("doNotModifyOutput", false)
		if !doNotModifyOutput {
			// Where the output will be written for this loop
			if confPath == "" || confPath == "stdout" {
				p.OutputFileDesc = os.Stdout
				p.OutputFile = "stdout"
			} else {
				// We check if the file is already open
				// If not, open it
				if p.OutputFile != confPath {
					// Close previous file if it's a file and not stdout
					if fileDesc, ok := p.OutputFileDesc.(*os.File); ok && fileDesc != os.Stdout {
						fileDesc.Close()
					}
					file, err := os.OpenFile(confPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
					if err != nil {
						queryData = QueryData{
							Message:    fmt.Sprintf("Error opening file %s: %s", confPath, err.Error()),
							StatusCode: 2,
						}
					} else {
						p.OutputFileDesc = file
						p.OutputFile = confPath
					}
				} else if fileDesc, ok := p.OutputFileDesc.(*os.File); ok && fileDesc != os.Stdout {
					// If the file is already open and it's a file (not stdout), we seek to the end
					// just to be sure to append the result
					fileDesc.Seek(0, 2)
				}
			}
			tempOutput = p.OutputFileDesc
		}

		// We check also if the output must be displayed only once somewhere
		/* if p.Config.GetString("onceOutputFile", "") != "" {
			// We open the file
			file, err := os.OpenFile(p.Config.GetString("onceOutputFile", ""), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
			if err != nil {
				queryData = QueryData{
					Message:    fmt.Sprintf("Error opening file %s: %s", p.Config.GetString("onceOutputFile", ""), err.Error()),
					StatusCode: 2,
				}

			} else {
				tempOutput = file
				// We close the file after writing to it
				tempOutputMustClose = true
			}
		} */

		// If the result is nil, we print the message
		if queryData.Result == nil {
			switch {
			case queryData.StatusCode <= 0:
				writeSuccessMessage(queryData.Message, tempOutput)
			case queryData.StatusCode == 1:
				writeWarningMessage(queryData.Message, tempOutput)
			default:
				writeErrorMessage(queryData.Message, tempOutput)
			}
		} else {
			// Create an output table and print it to the specified output
			table := outputTable{
				Writer: tempOutput,
				Type:   outputTableTypePretty,
			}

			// Check if the output mode is one of the specified
			if mode, ok := formatName[queryData.Config.GetString("outputMode", "")]; ok {
				table.Type = mode
			}

			// Write the SQL rows to the output
			err := table.WriteSQLRows(queryData.Result)
			if err != nil {
				writeErrorMessage(err.Error(), tempOutput)
			}

			err = table.Close()
			if err != nil {
				writeErrorMessage(fmt.Sprintf("Error closing table: %s", err.Error()), tempOutput)
			}

		}

		// Run all the post exec queries
		for _, postExec := range queryData.PostExec {
			_, err := queryData.DB.Exec(postExec)
			if err != nil {
				// Skip the errors for "nu such table" because they're expected
				// when a create table pre exec query fails
				// The error about why the table doesn't exist is already printed
				// so we don't need to print that we cannot drop it
				if !strings.Contains(err.Error(), "no such table:") {
					fmt.Fprintf(tempOutput, "Error running post exec query: %s\n", err.Error())
				}
			}
		}

		// We print a newline to separate the queries
		// unless it's the last query
		if i != len(queries)-1 {
			fmt.Fprintln(tempOutput)
		}

		/* // If the output must be closed, we close it
		if tempOutputMustClose {
			tempOutput.Close()
			// We set the output to the former output
			if p.OutputFileDesc != nil {
				tempOutput = p.OutputFileDesc
			} else {
				tempOutput = os.Stdout
			}
			p.Config.SetString("onceOutputFile", "")
		} */

	}

	return false
}

func writeSuccessMessage(message string, output io.Writer) {
	renderer := lipgloss.NewRenderer(output)
	success := renderer.NewStyle().Bold(true).Foreground(lipgloss.Color("35"))
	io.WriteString(output, success.Render(message))
	fmt.Fprintln(output)
}

func writeWarningMessage(message string, output io.Writer) {
	renderer := lipgloss.NewRenderer(output)
	warning := renderer.NewStyle().Bold(true).Foreground(lipgloss.Color("214"))
	io.WriteString(output, warning.Render(message))
	fmt.Fprintln(output)
}

func writeErrorMessage(message string, output io.Writer) {
	renderer := lipgloss.NewRenderer(output)
	errorS := renderer.NewStyle().Bold(true).Foreground(lipgloss.Color("160"))
	io.WriteString(output, errorS.Render(message))
	fmt.Fprintln(output)
}

func (p *shell) InputQuery() string {
	prompt := prompt.New(func(s string) {},
		prompt.WithHistory(p.History),
		prompt.WithPrefixTextColor(prompt.Fuchsia), prompt.WithKeyBind(
			prompt.KeyBind{
				Key: prompt.ControlC,
				Fn: func(p *prompt.Prompt) (rerender bool) {
					// Set .quit as the query
					p.Buffer().InsertTextMoveCursor(".quit", 0, 0, true)
					return true
				},
			},
			prompt.KeyBind{
				Key: prompt.ControlD,
				Fn: func(p *prompt.Prompt) (rerender bool) {
					// Set .exit as the query
					p.Buffer().InsertTextMoveCursor(".exit", 0, 0, true)
					return true
				},
			},
		),
		prompt.WithExecuteOnEnterCallback(func(prompt *prompt.Prompt, indentSize int) (indent int, execute bool) {
			// We trim the query
			query := strings.TrimSpace(prompt.Buffer().Text())

			// If the query is empty, we don't run it and consider the user wants to start with a newline
			if query == "" {
				return 0, false
			}

			// If the query starts with a dot, or with a backslash, we consider it as a command
			// And commands execute on enter
			if strings.HasPrefix(query, ".") || strings.HasPrefix(query, "\\") {
				return 0, true
			}

			// If the query ends with a semicolon, we consider it as a query
			if strings.HasSuffix(query, ";") {
				return 0, true
			}

			// Otherwise, we consider it as a multiline query that is not finished
			return 0, false
		}),
		prompt.WithPrefix("anyquery> "),
	)

	sqlQuery := prompt.Input()
	// We add the query to the history
	p.History = append(p.History, sqlQuery)

	return sqlQuery
}

// Split a query by the delimiter ; unless it is inside a string
// but also split by dot command (e.g. .exit) and slash command (e.g. \dt)
func splitMultipleQuery(sqlQuery string) []string {
	// isDotCommand => true if the query is a dot command or a slash command
	// isEscaped => true if the character is escaped (in a string '' or "")
	// mustResetWord => true if we find a query that can be appened to queries
	var isDotCommand, isEscapedSimpleQuote, isEscapedDoubleQuote, isMultiLineComment, isSingleLineComment, mustResetWord bool

	queries := make([]string, 0)
	// The index character of the current query
	// Reset to 0 when a new query is found
	currentPosition := 0

	tempQuery := strings.Builder{}

	for i, c := range sqlQuery {
		switch c {
		case ' ', '\t', '\r':
			// If we find a space and we are at the beginning of the query
			// we ignore it
			if currentPosition == 0 {
				continue
			}
		case '\\':
			// If we find a backslash and we are at the beginning of the query
			// we can assume that it is a slash command
			if i == 0 {
				isDotCommand = true
			}

		case '/':
			if len(sqlQuery) > i+1 && sqlQuery[i+1] == '*' {
				isMultiLineComment = true
			}

		case '*':
			// If we find a star and we are in a slash command
			// we can assume that it is the end of the slash command
			if isMultiLineComment && len(sqlQuery) > i+1 &&
				sqlQuery[i+1] == '/' {
				isMultiLineComment = false
			}
		case '-':
			// If we find a dash and we are at the beginning of the query
			// we can assume that it is a single line comment
			if i == 0 && len(sqlQuery) > i+1 && sqlQuery[i+1] == '-' {
				isSingleLineComment = true
			}

		case '.':
			// If we find a dot and we are at the beginning of the query
			// we can assume that it is a dot command
			if currentPosition == 0 {
				isDotCommand = true
			}

		case '\'':
			// SQLite uses SQL syntax to escape single quotes
			// so it doubles the single quote
			// Therefore, we can simply invert the value of isEscapedSimpleQuote
			isEscapedSimpleQuote = !isEscapedSimpleQuote

		case '"':
			// Same as above but for double quotes
			isEscapedDoubleQuote = !isEscapedDoubleQuote

		case ';':
			// If we find a semicolon and we are not in a string
			// we can assume that it is the end of the query
			//
			// It could be simplified as mustResetWord = condition
			// but I prefer to keep it like this for readability
			if !isEscapedSimpleQuote && !isEscapedDoubleQuote && !isMultiLineComment && !isSingleLineComment {
				mustResetWord = true
			}

		case '\n':
			// If we find a newline at the beginning of the query
			// we ignore it
			if currentPosition == 0 {
				continue
			}

			// If we find a newline, we're not in a string
			// and it's a dot command, it means that the query is finished
			if !isEscapedSimpleQuote && !isEscapedDoubleQuote && !isMultiLineComment && !isSingleLineComment && isDotCommand {
				mustResetWord = true
			}

			if isSingleLineComment {
				isSingleLineComment = false
			}

		}

		// We append the character to the query
		// unless it is a newline and it is a dot command
		//
		// Or it's a ; and we are not in a string
		if !((c == '\n' && isDotCommand) || (c == ';' && !isEscapedSimpleQuote && !isEscapedDoubleQuote && !isMultiLineComment && !isSingleLineComment)) {
			tempQuery.WriteRune(c)
		}
		currentPosition++

		// If we must reset the word, we append the query to the queries
		if mustResetWord {
			// Trim the whitespace at the beginning and the end of the query
			// and append it to the queries
			trimmed := strings.TrimSpace(tempQuery.String())
			queries = append(queries, trimmed)
			tempQuery.Reset()
			currentPosition = 0
			// Reset the flags
			isDotCommand = false
			mustResetWord = false
			isEscapedSimpleQuote = false
			isEscapedDoubleQuote = false
			isMultiLineComment = false
			isSingleLineComment = false
		}

	}

	// If the last query is not empty, we append it to the queries
	if tempQuery.Len() > 0 {
		trimmed := strings.TrimSpace(tempQuery.String())
		queries = append(queries, trimmed)
	}

	return queries
}
