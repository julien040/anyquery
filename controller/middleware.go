// This file defines middlewares to be used in the pipeline
// defined in pipeline.go
package controller

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/julien040/anyquery/namespace"
	"github.com/julien040/go-ternary"
	"vitess.io/vitess/go/vt/sqlparser"
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
	case "mode":
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

	case "once":
		if len(args) == 0 {
			queryData.Message = "No output file provided"
			queryData.StatusCode = 2
		} else {
			queryData.Config.SetString("onceOutputFile", args[0])
			// We don't set the message because it would be outputted
			// to the file
		}

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
		// If we wan't parse the query, we just pass it
		return true
	}
	if queryType == sqlparser.StmtShow {
		queryData.SQLQuery, queryData.Args = namespace.RewriteShowStatement(stmt.(*sqlparser.Show))
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
