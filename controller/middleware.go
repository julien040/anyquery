// This file defines middlewares to be used in the pipeline
// defined in pipeline.go
package controller

import (
	"fmt"
	"math/rand/v2"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/julien040/anyquery/namespace"
	"github.com/julien040/anyquery/other/prql"
	"github.com/julien040/anyquery/other/sqlparser"
	"github.com/julien040/go-ternary"
	"github.com/runreveal/pql"
)

func middlewareDotCommand(queryData *QueryData) bool {
	// Check if dot command are enabled
	if !queryData.Config.GetBool("dot-command", false) {
		return true
	}

	// Check if the query is a dot command
	query := queryData.SQLQuery

	// Ensure the trimmed query starts with a dot
	// If not, we skip to the next middleware
	if !strings.HasPrefix(strings.Trim(query, " "), ".") {
		return true
	}

	command, args := parseDotFunc(query)

	switch strings.ToLower(command) {
	case "cd":
		// Get the path
		if len(args) == 0 {
			queryData.Message = "No path provided"
			queryData.StatusCode = 2
		} else {
			err := os.Chdir(args[0])
			if err != nil {
				queryData.Message = err.Error()
				queryData.StatusCode = 2
			} else {
				queryData.Message = "Changed directory to " + args[0]
				queryData.StatusCode = 0
			}
		}

	/* case "columns":
	queryData.Config.SetString("outputMode", "columns") */

	case "databases", "database":
		queryData.SQLQuery = "SELECT name FROM pragma_database_list;"
		return true

	case "help":
		queryData.Message = "Documentation available at https://docs.anyquery.dev"
		queryData.StatusCode = 0

	case "indexes", "index":
		queryData.SQLQuery = "SELECT * FROM pragma_index_list;"

	case "log":
		if len(args) == 0 {
			queryData.Config.SetBool("logEnabled", false)
		} else {
			queryData.Config.SetString("logFile", args[0])
			queryData.Config.SetBool("logEnabled", true)
		}

	case "maxrows":
		if len(args) == 0 {
			queryData.Message = "No maxrows provided"
			queryData.StatusCode = 2
		} else {
			// Parse the maxrows
			val, err := strconv.Atoi(args[0])
			if err != nil {
				queryData.Message = err.Error()
				queryData.StatusCode = 2
			} else {
				queryData.Config.SetInt("maxRows", val)
			}
		}
	case "mode", "format":
		if len(args) == 0 {
			queryData.Message = "No mode provided"
			queryData.StatusCode = 2
		} else {
			// Check if the mode is valid
			if _, ok := formatName[args[0]]; !ok {
				queryData.Message = fmt.Sprintf("Invalid mode %s", args[0])
				queryData.StatusCode = 2
			} else {
				queryData.Config.SetString("outputMode", args[0])
				queryData.Message = fmt.Sprintf("Output mode set to %s", args[0])
				queryData.StatusCode = 0
			}
		}
	case "json":
		queryData.Config.SetString("outputMode", "json")
		queryData.Message = "Output mode set to JSON"
		queryData.StatusCode = 0

	case "jsonl":
		queryData.Config.SetString("outputMode", "jsonl")
		queryData.Message = "Output mode set to JSONL"
		queryData.StatusCode = 0

	case "csv":
		queryData.Config.SetString("outputMode", "csv")
		queryData.Message = "Output mode set to CSV"
		queryData.StatusCode = 0

	/* case "once":
	if len(args) == 0 {
		queryData.Message = "No output file provided"
		queryData.StatusCode = 2
	} else {
		queryData.Config.SetString("onceOutputFile", args[0])
		// We don't set the message because it would be outputted
		// to the file
	} */

	case "output":
		if len(args) == 0 {
			if queryData.Config.GetString("outputFile", "") != "" {
				queryData.Message = "Output now falls back to stdout"
				queryData.Config.SetString("outputFile", "")
			} else {
				queryData.Message = "No output file provided"
				queryData.StatusCode = 2
			}
		} else {
			queryData.Config.SetString("outputFile", args[0])
			// We don't set the message because it would be outputted
			// to the file
		}

	case "print":
		queryData.Message = strings.Join(args, " ")

	case "schema":
		if len(args) == 0 {
			queryData.SQLQuery = "SELECT sql FROM sqlite_schema"
		} else {
			queryData.SQLQuery = "SELECT sql FROM sqlite_schema WHERE tbl_name = ?"
			queryData.Args = append(queryData.Args, args[0])
		}

		return true

	case "separator":
		if len(args) == 0 {
			queryData.Message = "No separator provided"
			queryData.StatusCode = 2
		} else if len(args) == 1 {
			queryData.Config.SetString("separatorColumn", args[0])
		} else {
			queryData.Config.SetString("separatorColumn", args[0])
			queryData.Config.SetString("separatorRow", args[1])
		}

	case "shell", "system":
		// We run the command
		if len(args) == 0 {
			queryData.Message = "No command provided"
			queryData.StatusCode = 2
		} else {
			command := args[0]
			args = args[1:]
			cmd := exec.Command(command, args...)
			// Buffer the output
			output, err := cmd.CombinedOutput()
			if err != nil {
				queryData.Message = err.Error()
				queryData.StatusCode = 2
			} else {
				queryData.Message = string(output)
				queryData.StatusCode = 0
			}
		}
	case "tables", "table":
		queryData.SQLQuery = "SELECT name FROM pragma_table_list UNION SELECT name FROM pragma_module_list" +
			" WHERE name NOT LIKE 'fts%' AND name NOT LIKE 'rtree%'"
		return true
	case "languages", "language":
		if len(args) == 0 {
			queryData.Message = "Switching back to SQL"
			queryData.Config.SetString("language", "")
			queryData.StatusCode = 0
		} else {
			switch strings.ToLower(args[0]) {
			case "sql":
				queryData.Config.SetString("language", "")
				queryData.Message = "Switched back to SQL"

			case "prql":
				queryData.Config.SetString("language", "prql")
				queryData.Message = "Switched to PRQL"
				queryData.StatusCode = 0
			case "pql":
				queryData.Config.SetString("language", "pql")
				queryData.Message = "Switched to PQL"
				queryData.StatusCode = 0
			default:
				queryData.Message = "Unknown language"
				queryData.StatusCode = 2
			}
		}
	case "prql":
		if queryData.Config.GetString("language", "") == "prql" {
			queryData.Message = "Already using PRQL. Use .language to switch back to SQL"
			queryData.StatusCode = 1
		} else {
			queryData.Config.SetString("language", "prql")
			queryData.Message = "Switched to PRQL"
			queryData.StatusCode = 0
		}
	case "pql":
		if queryData.Config.GetString("language", "") == "pql" {
			queryData.Message = "Already using PQL. Use .language to switch back to SQL"
			queryData.StatusCode = 1
		} else {
			queryData.Config.SetString("language", "pql")
			queryData.Message = "Switched to PQL"
			queryData.StatusCode = 0
		}
	case "sql":
		queryData.Config.SetString("language", "")
		queryData.Message = "Switched back to SQL"
		queryData.StatusCode = 0
	}

	return false

}

// Return the command and the arguments of a dot command
func parseDotFunc(query string) (string, []string) {
	// We parse it by splitting it by spaces
	// and removing the first element
	// which is the dot command itself
	command := ""
	args := []string{}

	tempStr := strings.Builder{}
	for i := 1; i < len(query); i++ {
		if query[i] == ' ' {
			if command == "" {
				command = tempStr.String()
			} else {
				args = append(args, tempStr.String())
			}
			tempStr.Reset()
		} else {
			tempStr.WriteByte(query[i])
		}
	}

	// We add the last argument
	if command == "" {
		command = tempStr.String()
	} else {
		args = append(args, tempStr.String())
	}

	return command, args
}

// Rewrite the query
func middlewareMySQL(queryData *QueryData) bool {
	// Check if the query is a MySQL query
	if !queryData.Config.GetBool("mysql", false) {
		return true
	}

	queryType, stmt, err := namespace.GetQueryType(queryData.SQLQuery)
	if err != nil {
		// If we can't parse the query, we just pass it to the next middleware
		return true
	}
	if queryType == sqlparser.StmtShow {
		queryData.SQLQuery, queryData.Args = namespace.RewriteShowStatement(stmt.(*sqlparser.Show))
	} else if queryType == sqlparser.StmtExplain {
		// We rewrite the EXPLAIN/DESCRIBE statement
		if explain, ok := stmt.(*sqlparser.ExplainTab); ok {
			queryData.SQLQuery = "SELECT * FROM pragma_table_info(?);"
			queryData.Args = append(queryData.Args, explain.Table.Name.String())
		}
	}

	return true
}

func middlewareQuery(queryData *QueryData) bool {
	// Run the query on the database
	if queryData.DB == nil {
		return true
	}

	// Check if the query is empty
	// If we wouldn't do that, an empty query would hang forever
	if queryData.SQLQuery == "" {
		return true
	}
	// Run the pre-execution statements
	for i, preExec := range queryData.PreExec {
		_, err := queryData.DB.Exec(preExec)
		if err != nil {
			queryData.Message = fmt.Sprintf("Error in pre-execution statement %d: %s", i, err.Error())
			queryData.StatusCode = 2
			return false
		}
	}

	// Check whether the query must be run with Query or Exec
	// We need to check that because, for example, a CREATE VIRTUAL TABLE statement run with Query
	// will not return an error if it fails
	runWithQuery := true

	queryType, _, _ := namespace.GetQueryType(queryData.SQLQuery)
	switch queryType {
	case sqlparser.StmtSelect:
		runWithQuery = true
	case sqlparser.StmtUnknown:
		if strings.HasPrefix(strings.TrimSpace(strings.ToLower(queryData.SQLQuery)), "create virtual table") {
			runWithQuery = false
		} else {
			runWithQuery = true
		}
	default:
		runWithQuery = false
	}

	if runWithQuery {
		rows, err := queryData.DB.Query(queryData.SQLQuery, queryData.Args...)
		if err != nil {
			queryData.Message = err.Error()
			queryData.StatusCode = 2
			return false
		}
		queryData.Result = rows
	} else {
		res, err := queryData.DB.Exec(queryData.SQLQuery, queryData.Args...)
		if err != nil {
			queryData.Message = err.Error()
			queryData.StatusCode = 2
			return false
		}
		rowsAffected, err := res.RowsAffected()
		if err != nil {
			queryData.Message = "Successfully executed the query"
		} else {
			queryData.Message = fmt.Sprintf("Query executed successfully (%d %s affected)", rowsAffected, ternary.If(rowsAffected > 1, "rows", "row"))
		}
	}

	// Note: we can't run the post-execution statements here
	// because the result is not yet processed
	//
	// The post-execution statements are run at the end of the pipeline
	// after the output was printed
	return true
}

func middlewareSlashCommand(queryData *QueryData) bool {
	// Check if slash command are enabled
	if !queryData.Config.GetBool("slash-command", false) {
		return true
	}

	// Check if the query is a slash command
	query := queryData.SQLQuery

	// Ensure the trimmed query starts with a slash
	// If not, we skip to the next middleware
	trimmedQuery := strings.TrimSpace(query)
	if !strings.HasPrefix(trimmedQuery, "\\") {
		return true
	}

	splittedCommand := strings.Split(trimmedQuery, " ")
	commands := strings.ToLower(splittedCommand[0][1:])
	args := splittedCommand[1:]

	switch commands {
	case "l":
		queryData.SQLQuery = "SELECT name FROM pragma_database_list;"
		return true
	case "d":
		if len(args) == 0 {
			queryData.SQLQuery = "SELECT name FROM pragma_table_list UNION SELECT name FROM pragma_module_list" +
				" WHERE name NOT LIKE 'fts%' AND name NOT LIKE 'rtree%'"
		} else {
			queryData.SQLQuery = "SELECT name as Column, type as Type, '' as Collation, iif(\"notnull\" = 0, '', 'not null') as \"Null\"," +
				" dflt_value as \"Default\", pk as PrimaryKey FROM pragma_table_info(?)"

			queryData.Args = append(queryData.Args, args[0])
		}
		return true
	case "d+":
		if len(args) == 0 {
			queryData.SQLQuery = "SELECT * FROM pragma_table_list"
		} else {
			queryData.SQLQuery = "SELECT name as Column, type as Type, '' as Collation, iif(\"notnull\" = 0, '', 'not null') as \"Null\"," +
				" dflt_value as \"Default\", pk as PrimaryKey FROM pragma_table_info(?)"
			queryData.Args = append(queryData.Args, args[0])
		}
		return true
	case "di":
		queryData.SQLQuery = "SELECT * FROM pragma_index_list;"
		return true
	case "dt":
		queryData.SQLQuery = "SELECT name FROM pragma_table_list UNION SELECT name FROM pragma_module_list" +
			" WHERE name NOT LIKE 'fts%' AND name NOT LIKE 'rtree%'"
		return true
	case "dv":
		queryData.SQLQuery = "SELECT * FROM pragma_table_list WHERE type = 'view';"
		return true

	default:
		queryData.Message = fmt.Sprintf("Unknown command \\%s", commands)
		queryData.StatusCode = 2
	}

	return false

}

type table struct {
	name       string
	stringArgs []string
	position   int
	alias      string
}

const alphabet = "abcdefghijklmnopqrstuvwxyz"

func generateRandomString(size int) string {
	result := strings.Builder{}
	for i := 0; i < size; i++ {
		result.WriteByte(alphabet[rand.IntN(len(alphabet))])
	}
	return result.String()

}

var supportedTableFunctions = map[string]string{
	"read_json":    "json_reader",
	"read_csv":     "csv_reader",
	"read_parquet": "parquet_reader",
	"read_html":    "html_reader",
	"read_yaml":    "yaml_reader",
	"read_toml":    "toml_reader",
	"read_jsonl":   "jsonl_reader",
	"read_ndjson":  "jsonl_reader",
	"read_log":     "log_reader",
}

// Prefix the query like SELECT * FROM read_json with a CREATE VIRTUAL TABLE statement
func middlewareFileQuery(queryData *QueryData) bool {
	// # The problem
	//
	// To explain what this middleware does, let's take an example
	// SELECT * FROM read_json('file.json') WHERE name = 'John'
	//
	// file.json has a schema that is not known by the database
	// and SQLite does not support dynamic schema
	// Therefore, we need to register a virtual table that will read the file
	// and then we can query it
	//
	// # The solution
	//
	// The process is quite cumbersome. Therefore, we use a parser that detects
	// the table functions and replaces them with a random table name
	// Before running the query, we create a virtual table with the random name
	// and once the query is executed, we drop the table
	// That's a workaround around the limitation of SQLite

	// Parse the query
	parser, err := sqlparser.New(sqlparser.Options{
		MySQLServerVersion: "8.0.30",
	})

	if err != nil {
		return true
	}

	stmt, err := parser.Parse(queryData.SQLQuery)
	if err != nil {
		// We don't return any error here because an incompatible query with MySQL
		// might be compatible with SQLite
		//
		// If the query is not compatible with SQLite, the SQLite driver will return an error
		// so that's not an issue.
		return true
	}

	// Walk the AST and replace the table functions
	// with a CREATE VIRTUAL TABLE statement
	rewrite := func(cursor *sqlparser.Cursor) bool {
		// Get the table name
		tableFunction, ok := cursor.Node().(sqlparser.TableName)
		if !ok {
			return true
		}

		loweredName := strings.ToLower(tableFunction.Name.String())

		if !strings.HasPrefix(loweredName, "read_") {
			return true
		}

		tableName := generateRandomString(16)
		preExecBuilder := strings.Builder{}
		preExecBuilder.WriteString("CREATE VIRTUAL TABLE ")
		preExecBuilder.WriteString(tableName)
		preExecBuilder.WriteString(" USING ")
		if reader, ok := supportedTableFunctions[loweredName]; ok {
			preExecBuilder.WriteString(reader)
		} else {
			return true
		}

		if len(tableFunction.Args) > 0 {
			preExecBuilder.WriteString("(")
			for i, arg := range tableFunction.Args {
				if i > 0 {
					preExecBuilder.WriteString(", ")
				}
				preExecBuilder.WriteString(sqlparser.String(arg))
			}
			preExecBuilder.WriteString(")")
		}

		preExecBuilder.WriteString(";")

		queryData.PreExec = append(queryData.PreExec, preExecBuilder.String())

		queryData.PostExec = append(queryData.PostExec, "DROP TABLE "+tableName+";")

		cursor.Replace(sqlparser.NewTableName(tableName))

		return true
	}
	sqlparser.Rewrite(stmt, nil, rewrite)

	// In case it is a CREATE TABLE statement, we need to rewrite the select statement
	if createTable, ok := stmt.(*sqlparser.CreateTable); ok {
		if createTable.Select != nil {
			sqlparser.Rewrite(createTable.Select, nil, rewrite)
		}
	}

	// Deparse the query
	queryData.SQLQuery = sqlparser.String(stmt)

	/* // Extract the select statements
	selectStmts := extractSelectStmt(Result)
	if len(selectStmts) == 0 {
		return true
	}
	// So that we don't deparse if we don't need to
	modifiedTableCount := 0
	for _, selectStmt := range selectStmts {
		if selectStmt == nil {
			continue
		}
		if selectStmt.FromClause == nil {
			continue
		}
		tableFunctions := extractTableFunctions(selectStmt.FromClause)
		for _, tableFunction := range tableFunctions {
			// Check if the table function is a file module
			if !strings.HasPrefix(tableFunction.name, "read_") {
				continue
			}
			modifiedTableCount++
			// Replace the table function with a random one
			tableName := generateRandomString(16)
			preExecBuilder := strings.Builder{}
			preExecBuilder.WriteString("CREATE VIRTUAL TABLE ")
			preExecBuilder.WriteString(tableName)
			preExecBuilder.WriteString(" USING ")
			switch tableFunction.name {
			case "read_json":
				preExecBuilder.WriteString("json_reader")
			case "read_csv":
				preExecBuilder.WriteString("csv_reader")
			case "read_parquet":
				preExecBuilder.WriteString("parquet_reader")
			case "read_html":
				preExecBuilder.WriteString("html_reader")
			case "read_yaml":
				preExecBuilder.WriteString("yaml_reader")
			case "read_toml":
				preExecBuilder.WriteString("toml_reader")
			case "read_jsonl", "read_ndjson":
				preExecBuilder.WriteString("jsonl_reader")
			case "read_log":
				preExecBuilder.WriteString("log_reader")
			default:
				// If the user writes read_foo, and we don't have a reader for foo
				// we skip the table function
				continue
			}

			preExecBuilder.WriteString("(")
			for i, arg := range tableFunction.stringArgs {
				if i > 0 {
					preExecBuilder.WriteString(", ")
				}
				preExecBuilder.WriteRune('"')
				preExecBuilder.WriteString(arg)
				preExecBuilder.WriteRune('"')
			}
			preExecBuilder.WriteString(");")

			// Add the pre-execution statement
			queryData.PreExec = append(queryData.PreExec, preExecBuilder.String())

			// Add a post-execution statement to drop the table
			queryData.PostExec = append(queryData.PostExec, "DROP TABLE "+tableName+";")

			// Replace the table function with the new table name
			var tempTableName *pg_query.Node
			if tableFunction.alias == "" {
				tempTableName = pg_query.MakeSimpleRangeVarNode(tableName, int32(tableFunction.position))
			} else {
				tempTableName = pg_query.MakeFullRangeVarNode("", tableName, tableFunction.alias, int32(tableFunction.position))
			}
			selectStmt.FromClause[tableFunction.position] = tempTableName

		}
	}

	if modifiedTableCount > 0 {
		newQuery, err := pg_query.Deparse(Result)
		newQuery = regexp.MustCompile(`@[\s]+([a-zA-Z0-9_]+)`).ReplaceAllString(newQuery, "@$1")
		if err != nil {
			return true
		}
		queryData.SQLQuery = newQuery
	} */

	return true
}

func middlewarePRQL(query *QueryData) bool {
	if query.Config.GetString("language", "") != "prql" {
		return true
	}

	// Transform the query
	sqlQuery, messages := prql.ToSQL(query.SQLQuery)
	// If there are messages, the query is invalid
	if len(messages) > 0 {
		for _, message := range messages {
			query.Message = query.Message + message.Display + "\n"
			query.StatusCode = 2
		}
		return false
	}

	query.SQLQuery = sqlQuery
	return true
}

func middlewarePQL(query *QueryData) bool {
	if query.Config.GetString("language", "") != "pql" {
		return true
	}

	// Transform the query
	sqlQuery, err := pql.Compile(query.SQLQuery)
	if err != nil {
		query.Message = fmt.Sprintf("Error in PQL: %s", err.Error())
		query.StatusCode = 2
		return false
	}

	// Remove the semicolon at the end
	if strings.HasSuffix(sqlQuery, ";") {
		sqlQuery = sqlQuery[:len(sqlQuery)-1]
	}

	query.SQLQuery = sqlQuery
	return true
}
