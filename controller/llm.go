package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/adrg/xdg"
	"github.com/julien040/anyquery/controller/config/model"
	"github.com/julien040/anyquery/namespace"
	"github.com/julien040/anyquery/other/sqlparser"
	"github.com/julien040/anyquery/other/tunnel/client"
	"github.com/julien040/anyquery/rpc"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/spf13/cobra"
)

type describeTableBody struct {
	TableName string `json:"table_name"`
}

type executeQueryBody struct {
	Query string `json:"query"`
}

func Gpt(cmd *cobra.Command, args []string) error {

	// Find an open port the tunnel can listen on
	port, err := findOpenPort()
	if err != nil {
		return fmt.Errorf("failed to find an open port: %w", err)
	}

	// Open the configuration database
	cdb, queries, err := requestDatabase(cmd.Flags(), false)
	if err != nil {
		return fmt.Errorf("could not open the database: %w", err)
	}
	defer cdb.Close()

	host, _ := cmd.Flags().GetString("host")
	portUser, _ := cmd.Flags().GetInt("port")
	bindingAdress := fmt.Sprintf("127.0.0.1:%d", port)
	tunnelEnabled := true
	if host != "" && portUser != 0 {
		bindingAdress = fmt.Sprintf("%s:%d", host, portUser)
		tunnelEnabled = false
	}
	var tunnel *client.Tunnel

	// Get the tunnel from the database
	if tunnelEnabled {
		tunnel, err = getTunnel(queries, port)
		if err != nil {
			return fmt.Errorf("failed to get the HTTP tunnel: %w", err)
		}
	}

	// Open the database
	namespaceInstance, db, err := openUserDatabase(cmd, args)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}

	defer db.Close()

	// Create an HTTP server
	server := http.NewServeMux()
	server.HandleFunc("/list-tables", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		// List the plugins
		plugins := namespaceInstance.ListPluginsTables()
		w.Write([]byte("List of tables:\n"))
		for _, tables := range plugins {
			if tables.Description == "" {
				tables.Description = "No description"
			}
			w.Write([]byte(fmt.Sprintf("`%s` -- %s\n", tables.Name, tables.Description)))
		}

		// For each attached database, list the tables
		attachedDatabases := []string{}
		rows, err := db.Query("SELECT name from pragma_database_list")
		if err != nil {
			http.Error(w, "Failed to get attached databases", http.StatusInternalServerError)
			return
		}

		for rows.Next() {
			var dbName string
			if err := rows.Scan(&dbName); err != nil {
				http.Error(w, "Failed to get attached databases", http.StatusInternalServerError)
				return
			}

			attachedDatabases = append(attachedDatabases, dbName)
		}

		if rows.Err() != nil {
			http.Error(w, rows.Err().Error(), http.StatusInternalServerError)
			return
		}

		rows.Close()

		for _, dbName := range attachedDatabases {
			rows, err := db.Query(fmt.Sprintf("SELECT name FROM %s.sqlite_master WHERE type='table'", dbName))
			if err != nil {
				http.Error(w, "Failed to get table info", http.StatusInternalServerError)
			}

			for rows.Next() {
				var tableName string
				if err := rows.Scan(&tableName); err != nil {
					http.Error(w, "Failed to get table info", http.StatusInternalServerError)
					return
				}

				w.Write([]byte(fmt.Sprintf("`%s.%s` -- No description\n", dbName, tableName)))
			}

			if rows.Err() != nil {
				http.Error(w, rows.Err().Error(), http.StatusInternalServerError)
				return
			}

			rows.Close()
		}

		w.Header().Set("Content-Type", "text/plain")
		w.Header().Set("Cache-Control", "private, max-age=600")
	})
	server.HandleFunc("/describe-table", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		// Get the table name from the request
		body := describeTableBody{}

		// Decode the request body
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, fmt.Sprintf("Failed to decode the JSON body: %v", err), http.StatusBadRequest)
			return
		}

		tableName := body.TableName
		schema := "main"

		splitted := strings.SplitN(tableName, ".", 2)
		if len(splitted) == 2 {
			schema = strings.Trim(splitted[0], "`\" ")
			tableName = strings.Trim(splitted[1], "`\" ")
		} else {
			tableName = strings.Trim(tableName, "`\" ")
		}

		if strings.Contains(schema, ";") || strings.Contains(tableName, ";") || strings.Contains(schema, "`") || strings.Contains(tableName, "`") {
			http.Error(w, "Invalid table name", http.StatusBadRequest)
			return
		}

		// Get the table description
		rows, err := db.Query(fmt.Sprintf("SELECT name, type FROM  `%s`.pragma_table_info(?);", schema), tableName)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to get table info: %v", err), http.StatusInternalServerError)
			return
		}

		columns := []rpc.DatabaseSchemaColumn{}

		columnCount := 0
		for rows.Next() {
			columnCount++
			column := rpc.DatabaseSchemaColumn{}
			var columnType string
			if err := rows.Scan(&column.Name, &columnType); err != nil {
				http.Error(w, fmt.Sprintf("Failed to get column info: %v", err), http.StatusInternalServerError)
				return
			}

			switch {
			case strings.Contains(columnType, "INT"):
				column.Type = rpc.ColumnTypeInt
			case strings.Contains(columnType, "TEXT"), strings.Contains(columnType, "CHAR"),
				strings.Contains(columnType, "CLOB"):
				column.Type = rpc.ColumnTypeString
			case strings.Contains(columnType, "REAL"), strings.Contains(columnType, "FLOA"),
				strings.Contains(columnType, "DOUB"):
				column.Type = rpc.ColumnTypeFloat
			case strings.Contains(columnType, "BLOB"), strings.Contains(columnType, "BINARY"):
				column.Type = rpc.ColumnTypeBlob
			case strings.Contains(columnType, "BOOL"):
				column.Type = rpc.ColumnTypeBool
			case strings.Contains(columnType, "DATETIME"):
				column.Type = rpc.ColumnTypeDateTime
			case strings.Contains(columnType, "TIME"):
				column.Type = rpc.ColumnTypeTime
			case strings.Contains(columnType, "DATE"):
				column.Type = rpc.ColumnTypeDate
			case strings.Contains(columnType, "JSON"):
				column.Type = rpc.ColumnTypeJSON
			default:
				column.Type = rpc.ColumnTypeString
			}

			columns = append(columns, column)
		}
		if rows.Err() != nil {
			http.Error(w, fmt.Sprintf("Failed to get column info for table %s: %v", tableName, rows.Err()), http.StatusInternalServerError)
			return
		}

		if columnCount == 0 {
			http.Error(w, "Table not found", http.StatusBadRequest)
			return
		}

		// Get the table description
		desc, err := namespaceInstance.DescribeTable(tableName)
		if err != nil {
			desc = namespace.TableMetadata{
				Name:    tableName,
				Columns: columns,
				Insert:  true,
				Update:  true,
				Delete:  true,
			}
		}

		w.Header().Set("Content-Type", "text/plain")
		w.Header().Set("Cache-Control", "private, max-age=600")

		writeTableDescription(w, desc)
	})
	server.HandleFunc("/execute-query", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		body := executeQueryBody{}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, fmt.Sprintf("Failed to decode the JSON body: %v", err), http.StatusBadRequest)
			return
		}

		// Execute the query
		stmt, _, err := namespace.GetQueryType(body.Query)
		if err != nil { // If we can't determine the query type, we assume it's a SELECT
			stmt = sqlparser.StmtSelect
		}

		if stmt == sqlparser.StmtSelect {

			// Make a context that'll cancel the query after 40 seconds
			ctx, cancel := context.WithTimeout(context.Background(), 40*time.Second)
			defer cancel()

			rows, err := db.QueryContext(ctx, body.Query)
			if err != nil {
				http.Error(w, fmt.Sprintf("Failed to run the query: %v", err), http.StatusInternalServerError)
				return
			}

			defer rows.Close()

			columns, err := rows.Columns()
			if err != nil {
				http.Error(w, fmt.Sprintf("Failed to get the columns: %v", err), http.StatusInternalServerError)
				return
			}

			// Write the columns, and table data as a markdown table
			w.Write([]byte("|"))
			for _, column := range columns {
				w.Write([]byte(" "))
				w.Write([]byte(column))
				w.Write([]byte(" |"))
			}
			w.Write([]byte("\n|"))
			for _, column := range columns {
				w.Write([]byte(" "))
				for i := 0; i < len(column); i++ {
					w.Write([]byte("-"))
				}
				w.Write([]byte(" |"))
			}
			w.Write([]byte("\n"))

			for rows.Next() {
				values := make([]interface{}, len(columns))
				for i := range values {
					values[i] = new(interface{})
				}

				if err := rows.Scan(values...); err != nil {
					http.Error(w, fmt.Sprintf("Failed to scan the row: %v", err), http.StatusInternalServerError)
					return
				}

				w.Write([]byte("|"))
				for _, value := range values {
					w.Write([]byte(" "))
					unknown, ok := value.(*interface{})
					if ok && unknown != nil && *unknown != nil {
						switch parsed := (*unknown).(type) {
						case []byte:
							w.Write([]byte(fmt.Sprintf("%x", parsed)))
						case string:
							w.Write([]byte(fmt.Sprintf("%s", parsed)))
						case int64:
							w.Write([]byte(strconv.FormatInt(parsed, 10)))
						case float64:
							w.Write([]byte(strconv.FormatFloat(parsed, 'f', -1, 64)))
						case bool:
							if parsed {
								w.Write([]byte("true"))
							} else {
								w.Write([]byte("false"))
							}
						case time.Time:
							w.Write([]byte(parsed.Format(time.RFC3339)))

						default:
							w.Write([]byte(fmt.Sprintf("%v", *unknown)))
						}

					} else {
						w.Write([]byte("NULL"))
					}

					w.Write([]byte(" |"))
				}
				w.Write([]byte("\n"))
			}

			if rows.Err() != nil {
				http.Error(w, fmt.Sprintf("Failed to iterate over the rows %v", rows.Err()), http.StatusInternalServerError)
			}

		} else {
			res, err := db.Exec(body.Query)
			if err != nil {
				http.Error(w, fmt.Sprintf("Failed to execute the query: %v", err), http.StatusInternalServerError)
				return
			}

			rowsAffected, err := res.RowsAffected()
			if err != nil {
				http.Error(w, fmt.Sprintf("Failed to get the number of rows affected: %v", err), http.StatusInternalServerError)
				return
			}

			w.Write([]byte(fmt.Sprintf("Query executed, %d rows affected", rowsAffected)))
		}
	})

	// Start the tunnel in a separate goroutine
	if tunnelEnabled && tunnel != nil {
		go func() {
			if err := tunnel.Connect(); err != nil {
				fmt.Printf("failed to connect to the tunnel: %v\n", err)
			}
		}()
	}

	// Start the server
	if tunnel != nil {
		fmt.Printf("\n\nYour Anyquery ID is %s\n This is your bearer token that you must set in chatgpt.com (or similar tools)\n\n\n", tunnel.ID)
	} else {
		fmt.Printf("Local server listening on %s\n", bindingAdress)
		fmt.Printf("Your Anyquery ID is not available because the tunnel is disabled. Don't set the --host and --port flags to enable the tunnel\n")
		fmt.Println("Endpoints:")
		fmt.Println("GET /list-tables - List all the tables available")
		fmt.Println("POST /describe-table - Describe a table. Pass the table name in the body as a JSON object with the key 'table_name'")
		fmt.Println("POST /execute-query - Execute a query. Pass the query in the body as a JSON object with the key 'query'. Returns a markdown table for SELECT queries, and the number of rows affected for other queries")
	}
	if err := http.ListenAndServe(bindingAdress, server); err != nil {
		if tunnel != nil {
			tunnel.Close()
		}
		return fmt.Errorf("failed to start the HTTP server: %w", err)
	}

	return nil
}

func writeTableDescription(w io.Writer, desc namespace.TableMetadata) {
	w.Write([]byte(fmt.Sprintf("Table: %s\n", desc.Name)))
	if desc.PluginDescription != "" {
		desc.PluginDescription = " (" + desc.PluginDescription + ")"
	}
	if desc.Description == "" {
		desc.Description = "No description provided"
	}
	w.Write([]byte(fmt.Sprintf("Description: %s%s\n", desc.Description, desc.PluginDescription)))

	operationsSupported := []string{"select"}
	if desc.Insert {
		operationsSupported = append(operationsSupported, "insert")
	}

	if desc.Update {
		operationsSupported = append(operationsSupported, "update")
	}

	if desc.Delete {
		operationsSupported = append(operationsSupported, "delete")
	}

	w.Write([]byte(fmt.Sprintf("Supported operations for the table: %s\n\n", strings.Join(operationsSupported, ", "))))

	w.Write([]byte(fmt.Sprintf("Columns of the table %s (%d columns):\n", desc.Name, len(desc.Columns))))

	for i, column := range desc.Columns {
		columnTypeName := "TEXT"
		switch column.Type {
		case rpc.ColumnTypeBlob:
			columnTypeName = "BLOB"
		case rpc.ColumnTypeBool:
			columnTypeName = "BOOLEAN"
		case rpc.ColumnTypeFloat:
			columnTypeName = "REAL"
		case rpc.ColumnTypeInt:
			columnTypeName = "INTEGER"
		case rpc.ColumnTypeDate:
			columnTypeName = "DATE"
		case rpc.ColumnTypeDateTime:
			columnTypeName = "DATETIME"
		case rpc.ColumnTypeTime:
			columnTypeName = "TIME"
		case rpc.ColumnTypeJSON:
			columnTypeName = "JSON"
		}

		parameter := ""
		if column.IsParameter && column.IsRequired {
			parameter = " (required parameter)"
		} else if column.IsParameter {
			parameter = " (optional parameter)"
		}

		w.Write([]byte(fmt.Sprintf("%d. `%s` (type %s)%s - %s\n", i+1, column.Name, columnTypeName, parameter, column.Description)))
	}

	if len(desc.Examples) > 0 {
		w.Write([]byte("\nSQL examples:\n"))
		for i, example := range desc.Examples {
			w.Write([]byte(fmt.Sprintf("%d. %s\n\n", i+1, example)))
		}
	}
}

// Finds an open port so that the server can listen on it
func findOpenPort() (int, error) {
	// Start at 6969
	for i := 6969; i < 65535; i++ {
		_, err := net.Dial("tcp", fmt.Sprintf("127.0.0.1:%d", i))
		if err != nil {
			return i, nil
		}
	}
	return 0, fmt.Errorf("all ports are taken")
}

// Get the tunnel from the database
// or request a new one if it doesn't exist, or if the existing one is expired
//
// This tunnel will forward the HTTP requests from the GPTs to the local CLI
func getTunnel(configDB *model.Queries, localPort int) (*client.Tunnel, error) {
	// The data for the tunnel is available in entity-attribute-value table
	// The tunnel is identified by the entity "tunnel"
	// The attributes are "id", "auth_token" and "expires_at"

	// Get the tunnel from the database
	var id, authToken, expiresAt string
	attributes, err := configDB.GetEntityAttributes(context.Background(), "tunnel")
	if err != nil {
		return nil, fmt.Errorf("failed to get tunnel attributes from config database: %w", err)
	}

	for _, attr := range attributes {
		switch attr.Attribute {
		case "id":
			id = attr.Value
		case "auth_token":
			authToken = attr.Value
		case "expires_at":
			expiresAt = attr.Value
		}
	}

	// Get the current time + 1min to account for the dialing time
	// This is to prevent the tunnel from expiring while we are connecting to it
	currentTime := time.Now().Add(time.Minute).Format(time.RFC3339)

	if (id != "" && authToken != "" && expiresAt != "") && currentTime < expiresAt {
		// The tunnel exists and is not expired
		return client.NewTunnel(id, authToken, "127.0.0.1", localPort), nil
	}

	// The tunnel does not exist or is expired
	// Request a new tunnel
	tunnel, err := client.RequestTunnel()
	if err != nil {
		return nil, fmt.Errorf("failed to request a new tunnel from the API: %w", err)
	}

	// Save the tunnel to the database
	err = configDB.SetEntityAttributeValue(context.Background(), model.SetEntityAttributeValueParams{
		Entity:    "tunnel",
		Attribute: "id",
		Value:     tunnel.ID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to save the tunnel ID to the config database: %w", err)
	}

	err = configDB.SetEntityAttributeValue(context.Background(), model.SetEntityAttributeValueParams{
		Entity:    "tunnel",
		Attribute: "auth_token",
		Value:     tunnel.AuthToken,
	})

	if err != nil {
		return nil, fmt.Errorf("failed to save the tunnel auth token to the config database: %w", err)
	}

	err = configDB.SetEntityAttributeValue(context.Background(), model.SetEntityAttributeValueParams{
		Entity:    "tunnel",
		Attribute: "expires_at",
		Value:     tunnel.ExpiresAt,
	})

	if err != nil {
		return nil, fmt.Errorf("failed to save the tunnel expiration date to the config database: %w", err)
	}

	return client.NewTunnel(tunnel.ID, tunnel.AuthToken, "127.0.0.1", localPort), nil
}

func Mcp(cmd *cobra.Command, args []string) error {
	// Open the configuration database
	cdb, queries, err := requestDatabase(cmd.Flags(), false)
	if err != nil {
		return fmt.Errorf("could not open the database: %w", err)
	}
	defer cdb.Close()

	// Sometimes, the command might be launched from Claude Desktop,
	// which means the working directory is /
	// In this case, we need to change the working directory to the user's cache directory
	// because SQLite doesn't like to work in the root directory (fails to open the database, even if in memory)
	wd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get the working directory: %w", err)
	}

	if ((runtime.GOOS == "linux" || runtime.GOOS == "darwin") && wd == "/") || (runtime.GOOS == "windows" && wd == "C:\\") {
		os.Chdir(xdg.CacheHome)
	}

	// Open the database
	namespaceInstance, db, err := openUserDatabase(cmd, args)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}

	defer db.Close()

	// Catch the signals to close the database
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt)
	go func() {
		<-signalChan
		db.Close()
	}()

	s := server.NewMCPServer("Anyquery", "0.1.0")

	// Create the MCP server
	tool := mcp.NewTool("listTables", mcp.WithDescription("Lists all the tables available. When the user requests data, or wants an action (insert/update/delete), call this endpoint to check if a table corresponds to the user's request."))

	s.AddTool(tool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		response := strings.Builder{}

		// List the plugins
		plugins := namespaceInstance.ListPluginsTables()
		response.WriteString("List of tables:\n")
		for _, tables := range plugins {
			if tables.Description == "" {
				tables.Description = "No description"
			}
			response.WriteString(fmt.Sprintf("`%s` -- %s\n", tables.Name, tables.Description))
		}

		// For each attached database, list the tables
		attachedDatabases := []string{}
		rows, err := db.Query("SELECT name from pragma_database_list")
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to get attached databases from query: %v", err)), nil
		}

		for rows.Next() {
			var dbName string
			if err := rows.Scan(&dbName); err != nil {
				rows.Close()
				return mcp.NewToolResultError(fmt.Sprintf("failed to get attached databases while scanning: %v", err)), nil
			}
			attachedDatabases = append(attachedDatabases, dbName)
		}

		if rows.Err() != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to get attached databases after iteration: %v", rows.Err())), nil
		}

		rows.Close()
		for _, dbName := range attachedDatabases {
			rows, err := db.Query(fmt.Sprintf("SELECT name FROM %s.sqlite_master WHERE type='table'", dbName))
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("failed to get table info: %v", err)), nil
			}

			for rows.Next() {
				var tableName string
				if err := rows.Scan(&tableName); err != nil {
					rows.Close()
					return mcp.NewToolResultError(fmt.Sprintf("failed to get table info while iterating: %v", err)), nil
				}

				response.WriteString(fmt.Sprintf("`%s.%s` -- No description\n", dbName, tableName))
			}
		}

		return mcp.NewToolResultText(response.String()), nil

	})

	tool = mcp.NewTool("describeTable",
		mcp.WithDescription("Describes a table. It returns the columns of the table, the supported operations, and the SQL examples. Before calling executeQuery, you must call this endpoint for each table you want to interact with."),
		mcp.WithString("tableName", mcp.Required(), mcp.Description("The name of the table to describe")))

	s.AddTool(tool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		param, ok := request.Params.Arguments["tableName"]
		if !ok {
			return mcp.NewToolResultError("Missing tableName parameter"), nil
		}

		tableName, ok := param.(string)
		if !ok {
			return mcp.NewToolResultError("tableName must be a string"), nil
		}

		schema := "main"
		splitted := strings.SplitN(tableName, ".", 2)
		if len(splitted) == 2 {
			schema = strings.Trim(splitted[0], "`\" ")
			tableName = strings.Trim(splitted[1], "`\" ")
		} else {
			tableName = strings.Trim(tableName, "`\" ")
		}

		if strings.Contains(schema, ";") || strings.Contains(tableName, ";") || strings.Contains(schema, "`") || strings.Contains(tableName, "`") {
			return mcp.NewToolResultError("Invalid table name"), nil
		}

		rows, err := db.Query(fmt.Sprintf("SELECT name, type FROM  `%s`.pragma_table_info(?);", schema), tableName)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to get table info: %v", err)), nil
		}

		columns := []rpc.DatabaseSchemaColumn{}
		for rows.Next() {
			column := rpc.DatabaseSchemaColumn{}
			var columnType string
			if err := rows.Scan(&column.Name, &columnType); err != nil {
				rows.Close()
				return mcp.NewToolResultError(fmt.Sprintf("Failed to get column info: %v", err)), nil
			}

			switch {
			case strings.Contains(columnType, "INT"):
				column.Type = rpc.ColumnTypeInt
			case strings.Contains(columnType, "TEXT"), strings.Contains(columnType, "CHAR"), strings.Contains(columnType, "CLOB"):
				column.Type = rpc.ColumnTypeString
			case strings.Contains(columnType, "REAL"), strings.Contains(columnType, "FLOA"), strings.Contains(columnType, "DOUB"):
				column.Type = rpc.ColumnTypeFloat
			case strings.Contains(columnType, "BLOB"), strings.Contains(columnType, "BINARY"):
				column.Type = rpc.ColumnTypeBlob
			case strings.Contains(columnType, "BOOL"):
				column.Type = rpc.ColumnTypeBool
			case strings.Contains(columnType, "DATETIME"):
				column.Type = rpc.ColumnTypeDateTime
			case strings.Contains(columnType, "TIME"):
				column.Type = rpc.ColumnTypeTime
			case strings.Contains(columnType, "DATE"):
				column.Type = rpc.ColumnTypeDate
			case strings.Contains(columnType, "JSON"):
				column.Type = rpc.ColumnTypeJSON
			default:
				column.Type = rpc.ColumnTypeString
			}

			columns = append(columns, column)
		}

		if rows.Err() != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to get column info for table %s: %v", tableName, rows.Err())), nil
		}

		if len(columns) == 0 {
			return mcp.NewToolResultError("Table not found"), nil
		}

		desc, err := namespaceInstance.DescribeTable(tableName)
		if err != nil {
			desc = namespace.TableMetadata{
				Name:    tableName,
				Columns: columns,
				Insert:  true,
				Update:  true,
				Delete:  true,
			}
		}

		response := strings.Builder{}
		writeTableDescription(&response, desc)

		return mcp.NewToolResultText(response.String()), nil
	})

	tool = mcp.NewTool("executeQuery",
		mcp.WithDescription(`Executes a query on the SQL database. The query can be a SELECT, INSERT, UPDATE or DELETE query. Refer to the schema of each table to learn about the columns, their types, and whether the table supports INSERT, UPDATE or DELETE operations.
Before calling this endpoint, you must call listTables and describeTable for each table you want to interact with. This method will return the result of the query as a markdown table for SELECT queries, and the number of rows affected for INSERT, UPDATE and DELETE queries.

Anyquery is a SQL query engine for personal data. Using SELECT, INSERT, UPDATE, and DELETE, it can query and update data from various apps and SaaS used by its users (such as GitHub, Spotify, Notion, etc.). It runs on the local computer of the user, and you can access it using a few methods.
When a user requests some data, check if Anyquery can fulfill this need. Moreover, when the user requests a modification on an app, check if Anyquery has a table to do this update.

You have access to three methods:
- listTables
- describeTable
- executeQuery

listTables allows you to list all the SQL tables of Anyquery. When a user requests some data (e.g., check my emails) or wants you to update something (e.g., remind me to buy milk, add a new contact, etc.), run this method to check if a table fits the user's request.

For each table that fits the user's request, run a describeTable for the table. It will help you learn more about the schema and the required parameters for the table. Each field will have a description to help you constructing the query.

Once you have run listTables, and describeTable for each table of your query, you can run executeQuery. The SQL dialect is SQLite.
You must always follow the order listTables => describeTable => executeQuery. You cannot run a method before running the one that precedes it.

Anyquery uses the concept of table parameters like in SQLite. When you describeTable, you might come across fields that are specified as parameter, or required parameters.
Required parameters must be passed in the table argument for SELECT queries ("SELECT * FROM table(arg1, ..., argn)", in the WHERE condition for UPDATE/DELETE, and in "VALUES" for INSERTs.
For example, for the table github_repositories_from_user, you'll run "SELECT * FROM github_repositories_from_user('torvalds');" because the column user is set as a required parameter.

The omission of a required parameter will result in the "no query solution" error. If this error appears, double-check the parameters of the queried tables.

When you run describeTable, you might come across examples where the table name differs from the one you passed in the parameters of describeTable. You must still use the table name in the parameters.

You may uses JOIN, WHERE conditions, LIMIT clauses to reduce the amount of data sent.

When a user requests data and the filtered column is not a parameter, use "lower" on both sides of the "=" to make a non-case-sensitive comparison. (e.g. "SELECT * FROM myTable WHERE lower(lang) = lower('userValue');")
Use the "CONCAT" function to concat strings. Do not use "||".

Some columns might be returned as a JSON object or a JSON array. To extract a field of a JSON object, use the "->>" operator, and pass the JSON path (e.g. "SELECT col1 ->> ' $.myField.mySubField'").
For a JSON array, you may also use the "->>" operator. You can also create a new table with one row per element of the JSON array using the "json_each" table (e.g.  "SELECT j.value, a.* FROM myTable a, json_each(a.jsonArrayField)"); Finally, you can filter by the value of a JSON array using the "json_has" function (e.g. SELECT * FROM myTable WHERE json_has(jsonArrayField, 5) -- Returns rows where the JSON array jsonArrayField contains 5;").

Types handled by Anyquery are:
-  Text (a string)
-  Int
-  Float
-  Byte (returned as their hex representation)
-  DateTime (RFC3339)
-  Date (YYYY-MM-DD)
-  Time (HH:MM:SS)
-  JSON (can be an object or an array)

To reduce the amount of data transferred, please specify the column name in the SELECT statements. Avoid using the "*" wildcard for the columns as much as possible.

You have access to all the functions of the SQLite standard library such as the math functions, date functions, json functions, etc.

To handle datetime, use the "datetime(time-value, modifier, modifier, ...)" of SQLite. If no "time-value" is specified, it defaults to the current time. It supports several modifiers ± (e.g. "+ 7 years","- 2 months","+ 4 days","- 3 hours","+ 7 minutes", "+ 32 seconds".

`),
		mcp.WithString("query", mcp.Required(), mcp.Description("The SQL query to execute")))
	s.AddTool(tool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		// Get the table name from the request
		param, ok := request.Params.Arguments["query"]
		if !ok {
			return mcp.NewToolResultError("Missing query parameter"), nil
		}

		query, ok := param.(string)
		if !ok {
			return mcp.NewToolResultError("query must be a string"), nil
		}

		w := strings.Builder{}

		stmt, _, err := namespace.GetQueryType(query)
		if err != nil { // If we can't determine the query type, we assume it's a SELECT
			stmt = sqlparser.StmtSelect
		}

		if stmt == sqlparser.StmtSelect {

			// Make a context that'll cancel the query after 40 seconds
			ctx, cancel := context.WithTimeout(context.Background(), 40*time.Second)
			defer cancel()

			rows, err := db.QueryContext(ctx, query)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Failed to run the query: %v", err)), nil
			}

			defer rows.Close()

			columns, err := rows.Columns()
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Failed to get the columns: %v", err)), nil
			}

			// Write the columns, and table data as a markdown table
			w.Write([]byte("|"))
			for _, column := range columns {
				w.Write([]byte(" "))
				w.Write([]byte(column))
				w.Write([]byte(" |"))
			}
			w.Write([]byte("\n|"))
			for _, column := range columns {
				w.Write([]byte(" "))
				for i := 0; i < len(column); i++ {
					w.Write([]byte("-"))
				}
				w.Write([]byte(" |"))
			}
			w.Write([]byte("\n"))

			for rows.Next() {
				values := make([]interface{}, len(columns))
				for i := range values {
					values[i] = new(interface{})
				}

				if err := rows.Scan(values...); err != nil {
					return mcp.NewToolResultError(fmt.Sprintf("Failed to scan the row: %v", err)), nil
				}

				w.Write([]byte("|"))
				for _, value := range values {
					w.Write([]byte(" "))
					unknown, ok := value.(*interface{})
					if ok && unknown != nil && *unknown != nil {
						switch parsed := (*unknown).(type) {
						case []byte:
							w.Write([]byte(fmt.Sprintf("%x", parsed)))
						case string:
							w.Write([]byte(fmt.Sprintf("%s", parsed)))
						case int64:
							w.Write([]byte(strconv.FormatInt(parsed, 10)))
						case float64:
							w.Write([]byte(strconv.FormatFloat(parsed, 'f', -1, 64)))
						case bool:
							if parsed {
								w.Write([]byte("true"))
							} else {
								w.Write([]byte("false"))
							}
						case time.Time:
							w.Write([]byte(parsed.Format(time.RFC3339)))

						default:
							w.Write([]byte(fmt.Sprintf("%v", *unknown)))
						}

					} else {
						w.Write([]byte("NULL"))
					}

					w.Write([]byte(" |"))
				}
				w.Write([]byte("\n"))
			}

			if rows.Err() != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Failed to iterate over the rows %v", rows.Err())), nil
			}

		} else {
			res, err := db.Exec(query)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Failed to execute the query: %v", err)), nil
			}

			rowsAffected, err := res.RowsAffected()
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Failed to get the number of rows affected: %v", err)), nil
			}

			w.Write([]byte(fmt.Sprintf("Query executed, %d rows affected", rowsAffected)))
		}

		return mcp.NewToolResultText(w.String()), nil
	})

	// Start the server
	useStdio, _ := cmd.Flags().GetBool("stdio")

	if useStdio {
		return server.ServeStdio(s)
	} else {
		tunnelEnabled, _ := cmd.Flags().GetBool("tunnel")
		bindAddr := "127.0.0.1:8070"
		baseURL := "http://" + bindAddr

		if tunnelEnabled {
			// Find an open port the tunnel can listen on
			port, err := findOpenPort()
			if err != nil {
				return fmt.Errorf("failed to find an open port: %w", err)
			}

			// Get the tunnel from the database
			tunnel, err := getTunnel(queries, port)
			if err != nil {
				return fmt.Errorf("failed to get the HTTP tunnel: %w", err)
			}

			// Start the tunnel in a separate goroutine
			go func() {
				if err := tunnel.Connect(); err != nil {
					fmt.Printf("failed to connect to the tunnel: %v\n", err)
				}
			}()

			bindAddr = fmt.Sprintf("127.0.0.1:%d", port)
			baseURL = "https://mcp.anyquery.xyz/" + tunnel.ID
		} else {
			// Check if a domain is set
			domain, _ := cmd.Flags().GetString("domain")
			if domain != "" {
				baseURL = "https://" + domain
			}

			host, _ := cmd.Flags().GetString("host")
			port, _ := cmd.Flags().GetInt("port")
			if host != "" {
				bindAddr = fmt.Sprintf("%s:%d", host, port)
				baseURL = "http://" + bindAddr
			}
		}

		fmt.Printf("Model context protocol server listening on %s/sse\n", baseURL)

		sse := server.NewSSEServer(s, baseURL)
		// Catch the signals to gracefully close the server
		signalChanSSE := make(chan os.Signal, 1)
		signal.Notify(signalChanSSE, os.Interrupt)
		go func() {
			<-signalChanSSE
			sse.Shutdown(context.Background())
		}()

		return sse.Start(bindAddr)
	}

}
