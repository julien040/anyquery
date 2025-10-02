package controller

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"strings"
	"time"

	"github.com/adrg/xdg"
	"github.com/briandowns/spinner"
	"github.com/charmbracelet/lipgloss"
	"github.com/julien040/anyquery/controller/config/model"
	"github.com/julien040/anyquery/namespace"
	ws_tunnel "github.com/julien040/anyquery/other/websocket_tunnel/client"
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

var boxedStyle = lipgloss.NewStyle().Bold(false).Width(60).Foreground(lipgloss.Color("#f1f1f1")).Background(lipgloss.Color("#6f42c1")).Padding(1, 2, 1, 2).MarginTop(1).MarginBottom(1)

func listTablesLLM(namespaceInstance *namespace.Namespace, db *sql.DB, w io.Writer) error {
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
		return fmt.Errorf("failed to get attached databases: %w", err)
	}

	for rows.Next() {
		var dbName string
		if err := rows.Scan(&dbName); err != nil {
			return fmt.Errorf("failed to get attached databases: %w", err)
		}

		attachedDatabases = append(attachedDatabases, dbName)
	}

	if rows.Err() != nil {
		return fmt.Errorf("failed to get attached databases: %w", rows.Err())
	}

	rows.Close()

	for _, dbName := range attachedDatabases {
		rows, err := db.Query(fmt.Sprintf("SELECT name FROM %s.sqlite_master WHERE type='table'", dbName))
		if err != nil {
			return fmt.Errorf("failed to get table info: %w", err)
		}

		for rows.Next() {
			var tableName string
			if err := rows.Scan(&tableName); err != nil {
				return fmt.Errorf("failed to get table info: %w", err)
			}

			w.Write([]byte(fmt.Sprintf("`%s.%s` -- No description\n", dbName, tableName)))
		}

		if rows.Err() != nil {
			return fmt.Errorf("failed to get table info: %w", rows.Err())
		}

		rows.Close()
	}

	return nil
}

func describeTableLLM(namespaceInstance *namespace.Namespace, tableName string, db *sql.DB, w io.Writer) error {
	schema := "main"

	splitted := strings.SplitN(tableName, ".", 2)
	if len(splitted) == 2 {
		schema = strings.Trim(splitted[0], "`\" ")
		tableName = strings.Trim(splitted[1], "`\" ")
	} else {
		tableName = strings.Trim(tableName, "`\" ")
	}

	if strings.Contains(schema, ";") || strings.Contains(tableName, ";") || strings.Contains(schema, "`") || strings.Contains(tableName, "`") {
		return fmt.Errorf("invalid table name")
	}

	// Get the table description
	rows, err := db.Query(fmt.Sprintf("SELECT name, type FROM  `%s`.pragma_table_info(?);", schema), tableName)
	if err != nil {
		return fmt.Errorf("failed to get table info: %w", err)
	}

	columns := []rpc.DatabaseSchemaColumn{}

	columnCount := 0
	for rows.Next() {
		columnCount++
		column := rpc.DatabaseSchemaColumn{}
		var columnType string
		if err := rows.Scan(&column.Name, &columnType); err != nil {
			return fmt.Errorf("failed to get column info: %w", err)
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
		return fmt.Errorf("failed to get column info for table %s: %w", tableName, rows.Err())
	}

	if columnCount == 0 {
		return fmt.Errorf("table not found")
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

	writeTableDescription(w, desc)
	return nil
}

func executeQueryLLM(
	db *sql.DB,
	query string,
	w io.Writer,
) error {
	sh := shell{
		DB:             db,
		OutputFileDesc: w,
		Middlewares: []middleware{
			middlewareMySQL,
			middlewareFileQuery,
			middlewareQuery,
		},
		Config: middlewareConfiguration{
			"mysql":             true,
			"doNotModifyOutput": true, // Do not modify the output, just keep w
		},
	}

	// Run the query
	sh.Run(query)

	return nil
}

// Securely generates a random bearer token
func generateBearerToken() string {
	// Generate a random token
	token := make([]byte, 32)
	rand.Read(token)

	// Encode the token as a base64 string
	encodedToken := base64.StdEncoding.EncodeToString(token)

	return encodedToken
}

// Returns true if the request has a valid authorization header
func checkHTTPAuthorization(r *http.Request, bearerToken string) bool {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return false
	}

	authHeader = strings.TrimPrefix(authHeader, "Bearer ")
	return authHeader == bearerToken
}

func Gpt(cmd *cobra.Command, args []string) error {
	// Open the configuration database
	cdb, queries, err := requestDatabase(cmd.Flags(), false)
	if err != nil {
		return fmt.Errorf("could not open the database: %w", err)
	}
	defer cdb.Close()

	host, _ := cmd.Flags().GetString("host")
	portUser, _ := cmd.Flags().GetInt("port")
	tunnelEnabled := true
	if host != "" && portUser != 0 {
		tunnelEnabled = false
	}
	var tunnel *ws_tunnel.Tunnel

	// Get the tunnel from the database
	if tunnelEnabled {
		tunnel, err = getWsTunnel(queries)
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

	// Connect to the websocket server if tunnel is enabled
	if tunnelEnabled {
		// Connect to the websocket server
		fmt.Println("Connecting to the websocket server...")
		if err := tunnel.Connect(); err != nil {
			return fmt.Errorf("failed to connect to the websocket server: %w", err)
		}
		fmt.Println("Connected to the websocket server")

		fmt.Println(boxedStyle.Render("Anyquery is now running. When asked, pass", tunnel.ID, "as the anyquery ID to your LLM client (e.g. ChatGPT, TypingMind, etc.)\n\nID:", tunnel.ID, "\nThe tunnel will expire at", tunnel.ExpiresAt[:10], "UTC"))

		s := spinner.New(spinner.CharSets[31], 100*time.Millisecond)
		s.Prefix = "Waiting for requests "
		s.Start()
		defer s.Stop()

		updateSpinner := func(str string) {
			s.Prefix = str
			s.Restart()
		}

		for {
			req, err := tunnel.WaitRequest()
			if err != nil {
				return fmt.Errorf("failed to wait for websocket request. Check your internet connection: %w", err)
			}
			textRes := strings.Builder{}
			res := ws_tunnel.Response{
				RequestID: req.RequestID,
			}

			// Handle the request
			switch req.Method {
			case "list-tables":
				updateSpinner("Listing tables ")
				err := listTablesLLM(namespaceInstance, db, &textRes)
				if err != nil {
					res.Error = err.Error()
				}
			case "describe-table":
				updateSpinner("Describing table ")
				if len(req.Args) != 1 {
					res.Error = "missing table name"
				} else if _, ok := req.Args[0].(string); !ok {
					res.Error = "invalid table name"
				} else {
					err := describeTableLLM(namespaceInstance, req.Args[0].(string), db, &textRes)
					if err != nil {
						res.Error = err.Error()
					}
				}
			case "execute-query":
				updateSpinner("Executing query ")
				if len(req.Args) != 1 {
					res.Error = "missing query"
					continue
				}
				if _, ok := req.Args[0].(string); !ok {
					res.Error = "invalid query"
					continue
				}

				err := executeQueryLLM(db, req.Args[0].(string), &textRes)
				if err != nil {
					res.Error = err.Error()
				}
			default:
				res.Error = "unknown method. Supported methods are: list-tables, describe-table, execute-query. Perhaps you need to update Anyquery?"
			}

			// Send the response
			updateSpinner("Sending response ")
			res.Result = textRes.String()
			if err := tunnel.SendResponse(res); err != nil {
				return fmt.Errorf("failed to send response to the client: %w", err)
			}
			updateSpinner("Waiting for requests ")

		}
	}

	// This token will be used to authenticate the client
	// It must be supplied in the Authorization header of the request (prefixed with "Bearer ")
	//
	// The user can provide one using the environment variable ANYQUERY_AI_SERVER_BEARER_TOKEN
	bearerToken := generateBearerToken()

	envBearerToken := os.Getenv("ANYQUERY_AI_SERVER_BEARER_TOKEN")
	if envBearerToken != "" {
		bearerToken = envBearerToken
	}

	// Defaults to false
	// A flag to disable the authorization mechanism for locally bound servers
	noAuthHTTPFlag, _ := cmd.Flags().GetBool("no-auth")

	// Create an HTTP server if tunnel is disabled
	mux := http.NewServeMux()
	mux.HandleFunc("/list-tables", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Check the authorization header
		if !noAuthHTTPFlag && !checkHTTPAuthorization(r, bearerToken) {
			http.Error(w, "You must provide a valid authorization token prefixed with 'Bearer '", http.StatusUnauthorized)
			return
		}

		err := listTablesLLM(namespaceInstance, db, w)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "text/plain")
		w.Header().Set("Cache-Control", "private, max-age=600")
	})

	mux.HandleFunc("/describe-table", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Check the authorization header
		if !noAuthHTTPFlag && !checkHTTPAuthorization(r, bearerToken) {
			http.Error(w, "You must provide a valid authorization token prefixed with 'Bearer '", http.StatusUnauthorized)
			return
		}

		body := describeTableBody{}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, fmt.Sprintf("Failed to decode the JSON body: %v", err), http.StatusBadRequest)
			return
		}

		err := describeTableLLM(namespaceInstance, body.TableName, db, w)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "text/plain")
		w.Header().Set("Cache-Control", "private, max-age=600")
	})

	mux.HandleFunc("/execute-query", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Check the authorization header
		if !noAuthHTTPFlag && !checkHTTPAuthorization(r, bearerToken) {
			http.Error(w, "You must provide a valid authorization token prefixed with 'Bearer '", http.StatusUnauthorized)
			return
		}

		body := executeQueryBody{}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, fmt.Sprintf("Failed to decode the JSON body: %v", err), http.StatusBadRequest)
			return
		}

		err := executeQueryLLM(db, body.Query, w)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "text/plain")
		w.Header().Set("Cache-Control", "private, max-age=600")
	})

	fmt.Printf("Local server listening on %s:%d\n", host, portUser)
	if !noAuthHTTPFlag {
		fmt.Println("To authenticate, provide the authorization token in the Authorization header of the request (prefixed with 'Bearer ')")
		if envBearerToken != "" {
			fmt.Printf("Authorization token is supplied in the environment variable ANYQUERY_AI_SERVER_BEARER_TOKEN\n")
		} else {
			fmt.Printf("Authorization token: %s\n", bearerToken)
		}
	}
	fmt.Println("Methods:")
	fmt.Println("	GET /list-tables - List all the tables available")
	fmt.Println("	POST /describe-table - Describe a table. Pass the table name in the body as a JSON object with the key 'table_name'")
	fmt.Println("	POST /execute-query - Execute a query. Pass the query in the body as a JSON object with the key 'query'. Returns a markdown table for SELECT queries, and the number of rows affected for other queries")

	// Start the HTTP server
	err = http.ListenAndServe(fmt.Sprintf("%s:%d", host, portUser), mux)
	if err != nil {
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

// Get the tunnel from the database
// or request a new one if it doesn't exist, or if the existing one is expired
//
// This tunnel will forward the HTTP requests from the GPTs to the local CLI
func getWsTunnel(configDB *model.Queries) (*ws_tunnel.Tunnel, error) {
	// The data for the tunnel is available in entity-attribute-value table
	// The tunnel is identified by the entity "tunnel"
	// The attributes are "id", "auth_token" and "expires_at"

	// Get the tunnel from the database
	var id, authToken, expiresAt, serverURL string
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
		case "server_url":
			serverURL = attr.Value
		}
	}

	// Get the current time + 1min to account for the dialing time
	// This is to prevent the tunnel from expiring while we are connecting to it
	currentTime := time.Now().Add(time.Minute).Format(time.RFC3339)

	if (id != "" && authToken != "" && expiresAt != "" && serverURL != "") && currentTime < expiresAt {
		// The tunnel exists and is not expired
		return &ws_tunnel.Tunnel{
			ID:        id,
			AuthToken: authToken,
			ServerURL: serverURL,
			ExpiresAt: expiresAt,
		}, nil
	}

	// The tunnel does not exist or is expired
	// Request a new tunnel
	tunnel, err := ws_tunnel.RequestTunnel()
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

	err = configDB.SetEntityAttributeValue(context.Background(), model.SetEntityAttributeValueParams{
		Entity:    "tunnel",
		Attribute: "server_url",
		Value:     tunnel.ServerURL,
	})

	if err != nil {
		return nil, fmt.Errorf("failed to save the tunnel expiration date to the config database: %w", err)
	}

	return &ws_tunnel.Tunnel{
		ID:        tunnel.ID,
		AuthToken: tunnel.AuthToken,
		ServerURL: tunnel.ServerURL,
		ExpiresAt: tunnel.ExpiresAt,
	}, nil
}

func Mcp(cmd *cobra.Command, args []string) error {
	// Open the configuration database
	cdb, _, err := requestDatabase(cmd.Flags(), false)
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

	useStdio, _ := cmd.Flags().GetBool("stdio")
	tunnelEnabled, _ := cmd.Flags().GetBool("tunnel")

	authEnabled := false
	noAuthHTTPFlag, _ := cmd.Flags().GetBool("no-auth")
	// If the server is in HTTP mode, we need to enable the auth mechanism unless the user explicitly disables it
	if !noAuthHTTPFlag && !tunnelEnabled && !useStdio {
		authEnabled = true
	}

	bearerToken := generateBearerToken()
	if envBearerToken := os.Getenv("ANYQUERY_AI_SERVER_BEARER_TOKEN"); envBearerToken != "" {
		bearerToken = envBearerToken
	}

	s := server.NewMCPServer("Anyquery", "0.1.0")

	// Create the MCP server
	tool := mcp.NewTool("listTables", mcp.WithDescription("Lists all the tables available. When the user requests data, or wants an action (insert/update/delete), call this endpoint to check if a table corresponds to the user's request."))

	s.AddTool(tool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		if authEnabled {
			suppliedToken := request.Header.Get("Authorization")
			if suppliedToken == "" {
				return mcp.NewToolResultError("Missing authorization token"), nil
			}
			suppliedToken = strings.TrimPrefix(suppliedToken, "Bearer ")
			if suppliedToken != bearerToken {
				return mcp.NewToolResultError("Invalid authorization token"), nil
			}
		}

		response := strings.Builder{}
		err := listTablesLLM(namespaceInstance, db, &response)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to list tables: %v", err)), nil
		}

		return mcp.NewToolResultText(response.String()), nil

	})

	tool = mcp.NewTool("describeTable",
		mcp.WithDescription("Describes a table. It returns the columns of the table, the supported operations, and the SQL examples. Before calling executeQuery, you must call this endpoint for each table you want to interact with."),
		mcp.WithString("tableName", mcp.Required(), mcp.Description("The name of the table to describe")))

	s.AddTool(tool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		if authEnabled {
			suppliedToken := request.Header.Get("Authorization")
			if suppliedToken == "" {
				return mcp.NewToolResultError("Missing authorization token"), nil
			}

			suppliedToken = strings.TrimPrefix(suppliedToken, "Bearer ")
			if suppliedToken != bearerToken {
				return mcp.NewToolResultError("Invalid authorization token"), nil
			}
		}
		args := request.GetArguments()
		param, ok := args["tableName"]
		if !ok {
			return mcp.NewToolResultError("Missing tableName parameter"), nil
		}

		tableName, ok := param.(string)
		if !ok {
			return mcp.NewToolResultError("tableName must be a string"), nil
		}

		response := strings.Builder{}
		err := describeTableLLM(namespaceInstance, tableName, db, &response)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to describe table: %v", err)), nil
		}

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

Column names and table names with backticks. For example, SELECT `+"`"+`Équipe`+"`"+` FROM `+"`"+`my_table`+"`"+`;

To install Anyquery, the user must follow the tutorial at https://anyquery.dev/docs/#installation.

By default, Anyquery does not have any integrations. The user must visit https://anyquery.dev/integrations to find some integrations they might like and follow the instructions to add them.
`),
		mcp.WithString("query", mcp.Required(), mcp.Description("The SQL query to execute")))
	s.AddTool(tool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		if authEnabled {
			suppliedToken := request.Header.Get("Authorization")
			if suppliedToken == "" {
				return mcp.NewToolResultError("Missing authorization token"), nil
			}

			suppliedToken = strings.TrimPrefix(suppliedToken, "Bearer ")
			if suppliedToken != bearerToken {
				return mcp.NewToolResultError("Invalid authorization token"), nil
			}
		}
		// Get the table name from the request
		args := request.GetArguments()
		param, ok := args["query"]
		if !ok {
			return mcp.NewToolResultError("Missing query parameter"), nil
		}

		query, ok := param.(string)
		if !ok {
			return mcp.NewToolResultError("query must be a string"), nil
		}

		w := strings.Builder{}

		err := executeQueryLLM(db, query, &w)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to execute the query: %v", err)), nil
		}

		return mcp.NewToolResultText(w.String()), nil
	})

	// Start the server

	if useStdio {
		return server.ServeStdio(s)
	} else if tunnelEnabled {
		// Get the tunnel from the database
		/* tunnel, err := getWsTunnel(queries)
		if err != nil {
			return fmt.Errorf("failed to get the HTTP tunnel: %w", err)
		}

		// Start the tunnel in a separate goroutine
		if err := tunnel.Connect(); err != nil {
			fmt.Printf("failed to connect to the tunnel: %v\n", err)
		}

		// Catch the signals to gracefully close the server
		signalChanSSE := make(chan os.Signal, 1)
		signal.Notify(signalChanSSE, os.Interrupt)
		go func() {
			<-signalChanSSE
			tunnel.Close()
		}()

		// Handle the requests
		for {
			req, err := tunnel.WaitRequest()
			if err != nil {
				return fmt.Errorf("failed to wait for websocket request: %w", err)
			}

			fmt.Printf("Received request: %s\n", req.Method)

			var response strings.Builder
			switch req.Method {
			case "listTables":
				err := listTablesLLM(namespaceInstance, db, &response)
				if err != nil {
					return fmt.Errorf("failed to list tables: %w", err)
				}
			case "describeTable":
				if len(req.Args) != 1 {
					response.WriteString("Missing table name")
				} else if tableName, ok := req.Args[0].(string); !ok {
					response.WriteString("Invalid table name")
				} else {
					err := describeTableLLM(namespaceInstance, tableName, db, &response)
					if err != nil {
						return fmt.Errorf("failed to describe table: %w", err)
					}
				}
			case "executeQuery":
				if len(req.Args) != 1 {
					response.WriteString("Missing query")
				} else if query, ok := req.Args[0].(string); !ok {
					response.WriteString("Invalid query")
				} else {
					err := executeQueryLLM(db, query, &response)
					if err != nil {
						return fmt.Errorf("failed to execute query: %w", err)
					}
				}
			default:
				response.WriteString("Unknown method. Supported methods are: listTables, describeTable, executeQuery. Perhaps you need to update Anyquery?")
			}

			res := ws_tunnel.Response{
				RequestID: req.RequestID,
				Result:    response.String(),
			}

			if err := tunnel.SendResponse(res); err != nil {
				return fmt.Errorf("failed to send response to the client: %w", err)
			}
		} */

		return fmt.Errorf("tunnel is not supported. If this feature is needed, please open an issue on the GitHub repository.")

	} else {
		var baseURL string
		bindAddr := "127.0.0.1:7000"
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

		if authEnabled {
			fmt.Printf("Authentication enabled. Pass the token in the Authorization header of the request (prefixed with 'Bearer ')\n")
			if os.Getenv("ANYQUERY_AI_SERVER_BEARER_TOKEN") != "" {
				fmt.Printf("Authorization token is supplied in the environment variable ANYQUERY_AI_SERVER_BEARER_TOKEN\n")
			} else {
				fmt.Printf("Authorization token: %s\n", bearerToken)
			}
		}
		fmt.Printf("Model context protocol server listening on %s/sse\n", baseURL)

		sse := server.NewSSEServer(s, server.WithBaseURL(baseURL))
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
