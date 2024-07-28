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
	"github.com/julien040/go-ternary"
	pg_query "github.com/pganalyze/pg_query_go/v5"
	"github.com/runreveal/pql"
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

type tableFunction struct {
	name     string
	args     []string
	position int
	alias    string
}

// Extract the table functions from the query parsed by pg_query
func extractTableFunctions(fromClause []*pg_query.Node) []tableFunction {
	result := []tableFunction{}

	for ithTable, item := range fromClause {
		funcCall := item.GetRangeFunction()
		if funcCall == nil {
			continue
		}
		alias := ""
		if funcCall.Alias != nil {
			alias = funcCall.Alias.Aliasname
		}
		for _, function1 := range funcCall.Functions {
			// Get the function name
			nodeList, ok := function1.Node.(*pg_query.Node_List)
			if !ok {
				continue
			}

			for _, item := range nodeList.List.Items {
				funcCall := item.GetFuncCall()
				if funcCall != nil {
					if len(funcCall.Funcname) < 1 {
						continue
					}
					// Get the table name
					tableName := funcCall.Funcname[0].GetString_().Sval
					// Get args
					args := []string{}
					for _, arg := range funcCall.Args {
						// e.g. "foo", "bar"
						columnRef := arg.GetColumnRef()
						if columnRef != nil {
							args = append(args, columnRef.Fields[0].GetString_().Sval)
						}

						// e.g. 1, 'a", 1.0, true
						constRef := arg.GetAConst()
						if constRef != nil {
							svalStr := constRef.GetSval()
							if svalStr != nil {
								args = append(args, svalStr.Sval)
							}
							svalBool := constRef.GetBoolval()
							if svalBool != nil {
								if svalBool.Boolval {
									args = append(args, "true")
								} else {
									args = append(args, "false")
								}
							}
							svalInt := constRef.GetIval()
							if svalInt != nil {
								args = append(args, strconv.Itoa(int(svalInt.Ival)))
							}
							svalFloat := constRef.GetFval()
							if svalFloat != nil {
								args = append(args, svalFloat.Fval)
							}
						}

						// e.g. foo = bar
						exprRef := arg.GetAExpr()
						if exprRef != nil {
							leftSide := ""
							rightSide := ""
							// Get the left side of the expression
							left := exprRef.GetLexpr()
							if left != nil && left.GetColumnRef() != nil && len(left.GetColumnRef().Fields) > 0 {
								leftSide = left.GetColumnRef().Fields[0].GetString_().Sval
							} else if left != nil && left.GetAConst() != nil {
								if left.GetAConst() != nil {
									if left.GetAConst().GetIval() != nil {
										leftSide = strconv.Itoa(int(left.GetAConst().GetIval().Ival))
									} else if left.GetAConst().GetFval() != nil {
										leftSide = left.GetAConst().GetFval().Fval
									} else if left.GetAConst().GetSval() != nil {
										leftSide = left.GetAConst().GetSval().Sval
									} else if left.GetAConst().GetBoolval() != nil {
										if left.GetAConst().GetBoolval().Boolval {
											leftSide = "true"
										} else {
											leftSide = "false"
										}
									} else {
										leftSide = "NULL"
									}
								}
							}

							// Get the right side of the expression
							right := exprRef.GetRexpr()
							if right != nil && right.GetColumnRef() != nil && len(right.GetColumnRef().Fields) > 0 {
								rightSide = right.GetColumnRef().Fields[0].GetString_().Sval
							} else if right != nil && right.GetAConst() != nil {
								if right.GetAConst() != nil {
									if right.GetAConst().GetIval() != nil {
										rightSide = strconv.Itoa(int(right.GetAConst().GetIval().Ival))
									} else if right.GetAConst().GetFval() != nil {
										rightSide = right.GetAConst().GetFval().Fval
									} else if right.GetAConst().GetSval() != nil {
										rightSide = right.GetAConst().GetSval().Sval
									} else if right.GetAConst().GetBoolval() != nil {
										if right.GetAConst().GetBoolval().Boolval {
											rightSide = "true"
										} else {
											rightSide = "false"
										}
									} else {
										rightSide = "NULL"
									}

								}
							}

							args = append(args, leftSide+" = "+rightSide)

						}

					}

					result = append(result, tableFunction{
						name:     tableName,
						args:     args,
						position: ithTable,
						alias:    alias,
					})
				}
			}
		}
	}

	return result
}

const alphabet = "abcdefghijklmnopqrstuvwxyz"

func generateRandomString(size int) string {
	result := strings.Builder{}
	for i := 0; i < size; i++ {
		result.WriteByte(alphabet[rand.IntN(len(alphabet))])
	}
	return result.String()

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
	//
	// # Implementation issues
	//
	// The vitess's parser is not able to parse queries
	// with table functions like read_json() or read_csv()
	//
	// At first, I wanted to modify the parser to support
	// these functions, but it was too complicated
	// My knowledge of YACC is extremely limited
	//
	// As a temporary solution, I decided to parse these queries
	// with pg_query (it adds 6MB to the binary size so it's not ideal)
	// As the old saying goes, there is nothing more permanent than a temporary solution
	// I hope this is not the case here
	//
	// There is quite a lot of spaghetti code in this middleware to explore the AST
	// Couldn't find a better way to do it

	// Parse the query
	Result, err := pg_query.Parse(queryData.SQLQuery)
	if err != nil {
		return true
	}

	if Result == nil || len(Result.Stmts) == 0 || Result.Stmts[0].Stmt == nil {
		return true
	}

	selectStmt := Result.Stmts[0].Stmt.GetSelectStmt()
	if selectStmt == nil {
		// To handle INSERT INTO SELECT
		insertStmt := Result.Stmts[0].Stmt.GetInsertStmt()
		if insertStmt == nil {
			// To handle CREATE TABLE AS SELECT
			createTableStmt := Result.Stmts[0].Stmt.GetCreateTableAsStmt()
			if createTableStmt == nil {
				return true
			} else {
				selectStmt = createTableStmt.Query.GetSelectStmt()
				if selectStmt == nil {
					return true
				}
			}
		} else {
			selectStmt = insertStmt.SelectStmt.GetSelectStmt()
			if selectStmt == nil {
				return true
			}
		}

	}

	// Get the from clause
	tableFunctions := extractTableFunctions(selectStmt.FromClause)
	for _, tableFunction := range tableFunctions {
		// Check if the table function is a file module
		/* if tableFunction.name != "read_json" && tableFunction.name != "read_csv" && tableFunction.name != "read_parquet" &&
			tableFunction.name != "read_html" && tableFunction.name != "read_yaml" && tableFunction.name != "read_toml" &&
			tableFunction.name != "read_jsonl" && tableFunction.name != "read_ndjson" {
			continue
		} */
		if !strings.HasPrefix(tableFunction.name, "read_") {
			continue
		}

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
		default:
			// If the user writes read_foo, and we don't have a reader for foo
			// we skip the table function
			continue
		}

		preExecBuilder.WriteString("(")
		for i, arg := range tableFunction.args {
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

	newQuery, err := pg_query.Deparse(Result)
	if err != nil {
		return true
	}
	queryData.SQLQuery = newQuery

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
