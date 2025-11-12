package module

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"math/rand/v2"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/huandu/go-sqlbuilder"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/mattn/go-sqlite3"
)

var fetchClickHouseSchemaSQLQuery = `
SELECT DISTINCT
	lower(table_schema),
	lower(table_name),
	lower(column_name),
	lower(data_type)
FROM
	information_schema.columns
WHERE lower(table_schema) = lower({schema:String})
AND lower(table_name) = lower({table:String})
ORDER BY
	table_schema,
	table_name,
	ordinal_position ASC;
`

type ClickHouseModule struct {
	pooler map[string]*sql.DB // A pooler for each connection string
	mtx    *sync.RWMutex      // To protect the pooler from concurrent access
}

// Retrieve or create a connection from the pooler of the pooler
func (m *ClickHouseModule) GetDBConnection(connectionString string) (*sql.Conn, *clickhouse.Options, error) {
	m.mtx.RLock()
	if pool, ok := m.pooler[connectionString]; ok {
		m.mtx.RUnlock()
		conn, err := pool.Conn(context.Background())
		options, _ := clickhouse.ParseDSN(connectionString) // err == nil if the connection string is valid

		return conn, options, err
	}
	m.mtx.RUnlock()

	// Parse the DSN
	if connectionString == "" {
		return nil, nil, fmt.Errorf("connection string is empty")
	}

	options, err := clickhouse.ParseDSN(connectionString)
	if err != nil {
		return nil, nil, fmt.Errorf("error parsing the connection string: %v. Make sure it is a valid ClickHouse DSN (https://github.com/ClickHouse/clickhouse-go#dsn)", err)
	}

	pool := clickhouse.OpenDB(options)
	if pool == nil {
		return nil, nil, fmt.Errorf("error opening a connection to the database with the connection string: %v", connectionString)
	}

	m.mtx.Lock()
	m.pooler[connectionString] = pool
	m.mtx.Unlock()

	conn, err := pool.Conn(context.Background())
	if err != nil {
		return nil, nil, fmt.Errorf("error getting a connection from the pool: %v", err)
	}
	return conn, options, nil
}

type ClickHouseTable struct {
	connection       *sql.Conn
	tableName        string
	schema           []databaseColumn
	module           *ClickHouseModule
	connectionString string
}

type ClickHouseCursor struct {
	connection   *sql.Conn
	tableName    string
	schema       []databaseColumn
	rows         *sql.Rows
	exhausted    bool
	currentRow   []interface{}
	rowsReturned int64
	limit        int64
	query        SQLQueryToExecute
}

func (m *ClickHouseModule) Create(c *sqlite3.SQLiteConn, args []string) (sqlite3.VTab, error) {
	return m.Connect(c, args)
}

func (m *ClickHouseModule) TransactionModule() {}

func (v *ClickHouseModule) DestroyModule() {}

func (m *ClickHouseModule) Connect(c *sqlite3.SQLiteConn, args []string) (sqlite3.VTab, error) {
	// Init the structure
	if m.pooler == nil {
		m.pooler = make(map[string]*sql.DB)
	}
	if m.mtx == nil {
		m.mtx = &sync.RWMutex{}
	}

	// Fetch the arguments
	connectionString := ""
	table := ""
	schemaName := ""
	if len(args) >= 4 {
		connectionString = strings.Trim(args[3], "' \"") // Remove the quotes
	}
	if len(args) >= 5 {
		table = strings.Trim(args[4], "' \"") // Remove the quotes
	}

	params := []argParam{
		{"connection_string", &connectionString},
		{"connectionString", &connectionString},
		{"url", &connectionString},
		{"uri", &connectionString},
		{"dsn", &connectionString},
		{"data_source_name", &connectionString},
		{"connection", &connectionString},
		{"conn", &connectionString},
		{"table", &table},
		{"name", &table},
		{"table_name", &table},
		{"tableName", &table},
	}
	parseArgs(params, args)

	if connectionString == "" {
		return nil, fmt.Errorf("missing connection string argument. Check the validity of the arguments")
	}

	if table == "" {
		return nil, fmt.Errorf("missing table argument. Check the validity of the arguments")
	}

	// Open the database
	conn, opt, err := m.GetDBConnection(connectionString)
	if err != nil {
		return nil, fmt.Errorf("error opening a connection to the database: %v", err)
	}

	// Return the connection
	defer conn.Close()

	// Parse the tableName and split it into schema and table if needed
	schemaTable := strings.Split(table, ".")
	if len(schemaTable) > 1 {
		schemaName = schemaTable[0]
		table = schemaTable[1]

		// Remove any quotes, backticks or spaces around the schema and table names
		schemaName = strings.Trim(schemaName, "\" '`")
		table = strings.Trim(table, "\" '`")
	}

	if schemaName == "" {
		schemaName = "default"
		if opt.Auth.Database != "" {
			schemaName = opt.Auth.Database // Use the default database if no schema is provided
		}
	}

	// Fetch the schema for the table
	rows, err := conn.QueryContext(context.Background(), fetchClickHouseSchemaSQLQuery, clickhouse.Named("schema", schemaName), clickhouse.Named("table", table))
	if err != nil {
		return nil, fmt.Errorf("error fetching the schema for the table: %v", err)
	}
	defer rows.Close()

	// Iterate over the rows and create the schema
	internalSchema := []databaseColumn{}
	schema := strings.Builder{}
	schema.WriteString("CREATE TABLE x(\n")
	firstRow := true
row:
	for rows.Next() {
		var tableSchema, tableName, columnName, dataType string
		err = rows.Scan(&tableSchema, &tableName, &columnName, &dataType)
		if err != nil {
			return nil, fmt.Errorf("error scanning the schema: %v", err)
		}
		dataType = strings.TrimSpace(dataType)
		dataType = strings.ToLower(dataType)

		if strings.HasPrefix(dataType, "lowcardinality(") && strings.HasSuffix(dataType, ")") {
			dataType = strings.TrimPrefix(dataType, "lowcardinality(")
			dataType = strings.TrimSuffix(dataType, ")")
		}

		// If the column is nullable, it's type is nullable(type)
		// We therefore remove the "nullable" part
		if strings.HasPrefix(dataType, "nullable(") && strings.HasSuffix(dataType, ")") {
			dataType = strings.TrimPrefix(dataType, "nullable(")
			dataType = strings.TrimSuffix(dataType, ")")
		}

		columnType := "TEXT"
		typeSupported := false
		switch {
		case strings.HasPrefix(dataType, "int") || strings.HasPrefix(dataType, "uint"):
			columnType = "INTEGER"
			typeSupported = true
		case strings.HasPrefix(dataType, "float") || strings.HasPrefix(dataType, "double") || dataType == "real" ||
			dataType == "single" || strings.HasPrefix(dataType, "decimal"):
			columnType = "REAL"
			typeSupported = true

		case strings.Contains(dataType, "text") || strings.Contains(dataType, "char") || dataType == "string" ||
			strings.Contains(dataType, "blob") || strings.Contains(dataType, "binary") || strings.HasPrefix(dataType, "fixedstring") ||
			strings.HasPrefix(dataType, "enum") || strings.HasPrefix(dataType, "uuid") || dataType == "ipv4" ||
			dataType == "ipv6":
			columnType = "TEXT"
			typeSupported = true

		case strings.HasPrefix(dataType, "datetime"):
			columnType = "DATETIME"
			typeSupported = true

		case strings.HasPrefix(dataType, "date"):
			columnType = "DATE"
			typeSupported = true

		case strings.HasPrefix(dataType, "array") || strings.HasPrefix(dataType, "tuple") ||
			strings.HasPrefix(dataType, "map") || strings.HasPrefix(dataType, "json"):
			columnType = "JSON"
			typeSupported = true

		case dataType == "point" || dataType == "linestring" || dataType == "ring" ||
			dataType == "polygon" || dataType == "multipolygon" || dataType == "multilinestring":
			columnType = "TEXT"
			typeSupported = true

		case dataType == "bool":
			columnType = "BOOLEAN"
			typeSupported = true

		case strings.HasPrefix(dataType, "variant(") || dataType == "dynamic":
			columnType = "UNKNOWN"
			typeSupported = true

		case strings.HasPrefix(dataType, "nested("):
			continue row // Skip nested types, as we don't support them

		default:
			columnType = "UNKNOWN"
			typeSupported = false // Fail safe
		}

		localColumnName := transformSQLiteValidName(columnName)
		if !firstRow {
			schema.WriteString(",\n")
		}
		firstRow = false

		schema.WriteString(fmt.Sprintf("  \"%s\" %s", localColumnName, columnType))
		internalSchema = append(internalSchema, databaseColumn{
			Realname:   columnName,
			SQLiteName: localColumnName,
			Type:       columnType,
			RemoteType: dataType,
			Supported:  typeSupported,
		})
	}

	if rows.Err() != nil {
		return nil, fmt.Errorf("error iterating over the rows: %v", rows.Err())
	}

	schema.WriteString("\n)")

	if len(internalSchema) == 0 {
		return nil, fmt.Errorf("no columns found for the table")
	}

	// Declare the virtual table
	err = c.DeclareVTab(schema.String())
	if err != nil {
		return nil, fmt.Errorf("error declaring the virtual table: %v", err)
	}

	// Merge the schema with the table name
	table = fmt.Sprintf("`%s`.`%s`", schemaName, table)

	// Return the table instance
	return &ClickHouseTable{
		tableName:        table,
		schema:           internalSchema,
		module:           m,
		connectionString: connectionString,
	}, nil
}

func (t *ClickHouseTable) Open() (sqlite3.VTabCursor, error) {
	// Create a new connection for each cursor
	conn, _, err := t.module.GetDBConnection(t.connectionString)
	if err != nil {
		return nil, fmt.Errorf("error getting a new connection: %v", err)
	}

	values := make([]interface{}, len(t.schema))
	for i := range values {
		values[i] = new(interface{})
		switch t.schema[i].Type {
		case "INTEGER":
			if t.schema[i].RemoteType[0] == 'u' {
				// For unsigned integers, we use NullUint64
				values[i] = new(NullUint64) // ClickHouse does not have unsigned integers, so we use NullInt64
			} else {
				values[i] = new(sql.NullInt64)
			}
		case "REAL":
			values[i] = new(sql.NullFloat64)
		case "TEXT":
			if t.schema[i].RemoteType == "ipv4" || t.schema[i].RemoteType == "ipv6" {
				// For IPv4 and IPv6, we use a net.IP type
				values[i] = new(net.IP)
			} else {
				values[i] = new(sql.NullString)
			}
		case "BLOB":
			values[i] = new([]byte)
		case "DATE":
			values[i] = new(timeMySQL)
		case "DATETIME":
			values[i] = new(timeMySQL)
		default:
			values[i] = new(interface{})
		}
	}

	return &ClickHouseCursor{
		connection: conn,
		tableName:  t.tableName,
		schema:     t.schema,
		limit:      -1,
		currentRow: values,
	}, nil
}

func (t *ClickHouseTable) Disconnect() error {
	return t.Destroy()
}

func (t *ClickHouseTable) releaseConnection() {
	if t.connection != nil {
		t.connection.Close()
		t.connection = nil
	}
}

func (t *ClickHouseTable) Destroy() error {
	// Release the connection
	t.releaseConnection()
	return nil
}

// A function that will be called several times to check the best way to access the data.
// This function is called with different constraints and order by clauses
//
// To find the method, we will ask the database to explain the query and return the best method
func (t *ClickHouseTable) BestIndex(cst []sqlite3.InfoConstraint, ob []sqlite3.InfoOrderBy, info sqlite3.IndexInformation) (*sqlite3.IndexResult, error) {
	// Create the SQL query
	queryBuilder, limitCstIndex, offsetCstIndex, used := efficientConstructSQLQuery(cst, ob, t.schema, t.tableName, info.ColUsed, sqlbuilder.ClickHouse)
	queryBuilder.SetFlavor(sqlbuilder.ClickHouse)
	rawQuery, args := queryBuilder.Build()
	rawQuery += sqlQuerySuffix

	// Request a connection
	if t.connection == nil {
		conn, _, err := t.module.GetDBConnection(t.connectionString)
		if err != nil {
			return nil, fmt.Errorf("error getting a new connection for best index: %v", err)
		}
		t.connection = conn
	}
	defer t.releaseConnection()

	// Explain the query and ignore any error if the query is not valid
	// This might happen with MySQL-wire compatible databases that don't use the same syntax
	explainQuery := "EXPLAIN ESTIMATE " + rawQuery
	row := t.connection.QueryRowContext(context.Background(), explainQuery, args...)
	var db, table string
	var parts, rows, marks int64

	row.Scan(&db, &table, &parts, &rows, &marks)
	if row.Err() != nil {
		return nil, fmt.Errorf("error explaining the query: %v", row.Err())
	}

	// Reduce the cost if the query has a limit or offset
	if limitCstIndex != -1 {
		rows = -1
	}
	if offsetCstIndex != -1 {
		rows = -1
	}

	// Set alreadyOrdered flag if the requested ordered columns are all supported
	alreadyOrdered := true
	for _, o := range ob {
		if o.Column < 0 || o.Column >= len(t.schema) {
			alreadyOrdered = false
			break
		}
		alreadyOrdered = alreadyOrdered && t.schema[o.Column].Supported
	}

	query := &SQLQueryToExecute{
		Query:       rawQuery,
		Args:        args,
		LimitIndex:  limitCstIndex,
		OffsetIndex: offsetCstIndex,
		ColumnsUsed: info.ColUsed,
	}

	// Serialize the query as a JSON object
	serializedQuery, err := json.Marshal(query)
	if err != nil {
		return nil, fmt.Errorf("error serializing the query: %v", err)
	}

	return &sqlite3.IndexResult{
		Used:           used,
		IdxStr:         string(serializedQuery),
		EstimatedRows:  float64(rows),
		AlreadyOrdered: alreadyOrdered,
	}, nil
}

/*
type VTabUpdater interface {
    VTab
    Delete(any) error
    Insert(any, []any) (int64, error)
    Update(any, []any) error
    PartialUpdate() bool
} */

// DML related functions
func (t *ClickHouseTable) Insert(id any, vals []any) (int64, error) {
	builder := sqlbuilder.NewInsertBuilder()
	builder.InsertInto(t.tableName)
	cols := []string{}
	values := []interface{}{}
	for i, v := range vals {
		if v == nil {
			continue
		}
		cols = append(cols, t.schema[i].Realname)
		values = append(values, v)
	}
	builder.Cols(cols...)
	builder.Values(values...)
	builder.SetFlavor(sqlbuilder.ClickHouse)

	query, args := builder.Build()
	query = sqlQuerySuffix + query

	// Request a connection
	if t.connection == nil {
		conn, _, err := t.module.GetDBConnection(t.connectionString)
		if err != nil {
			return 0, fmt.Errorf("error getting a new connection for insert: %v", err)
		}
		t.connection = conn
	}
	defer t.releaseConnection()

	_, err := t.connection.ExecContext(context.Background(), query, args...)
	if err != nil {
		return 0, fmt.Errorf("error executing the insert query: %v", err)
	}

	return rand.Int64(), nil
}

func (t *ClickHouseTable) Update(id any, vals []any) error {
	return fmt.Errorf("update operation is not supported for ClickHouse tables")
}

func (t *ClickHouseTable) Delete(id any) error {
	return fmt.Errorf("delete operation is not supported for ClickHouse tables")
}

func (t *ClickHouseTable) PartialUpdate() bool {
	return false
}

func (t *ClickHouseCursor) resetCursor() error {
	if t.rows != nil {
		t.rows.Close()
		t.rows = nil
	}
	t.limit = -1
	t.rowsReturned = 0
	t.exhausted = false

	return nil
}

func (t *ClickHouseCursor) Filter(idxNum int, idxStr string, vals []interface{}) error {
	// Reset the cursor as Filter might be called multiple times
	err := t.resetCursor()
	if err != nil {
		return fmt.Errorf("error resetting the cursor: %v", err)
	}

	// Reconstruct the query and its arguments
	var query SQLQueryToExecute
	err = json.Unmarshal([]byte(idxStr), &query)
	if err != nil {
		return fmt.Errorf("error unmarshalling the query: %v", err)
	}

	// Set the query for the cursor
	t.query = query

	// Get the LIMIT AND OFFSET values
	// and remove them from the query so that we can pass these arguments to the query
	limit := int64(-1)
	offset := int64(-1)
	queryParams := []interface{}{}
	for i, c := range vals {
		if i == query.LimitIndex {
			limit = c.(int64)
		} else if i == query.OffsetIndex {
			offset = c.(int64)
		} else {
			queryParams = append(queryParams, c)
		}
	}

	// Add the LIMIT and OFFSET to the query
	if limit != -1 {
		query.Query += fmt.Sprintf(" LIMIT %d", limit)
		t.limit = limit
	}
	if offset != -1 {
		query.Query += fmt.Sprintf(" OFFSET %d", offset)
	}

	// Execute the query
	rows, err := t.connection.QueryContext(context.Background(), query.Query, queryParams...)
	if err != nil {
		return fmt.Errorf("error executing the query: %v", err)
	}

	t.rows = rows
	return t.Next()
}

func (t *ClickHouseCursor) Next() error {
	t.rowsReturned++
	if t.rows == nil {
		return fmt.Errorf("no rows to iterate over")
	}
	hasMoreRows := t.rows.Next()
	t.exhausted = !hasMoreRows
	if t.rows.Err() != nil {
		return fmt.Errorf("error iterating over the rows: %v", t.rows.Err())
	}
	if hasMoreRows {
		var err error

		dest := make([]interface{}, 0, len(t.schema))
		for i := range t.schema {
			// ColumnsUsed is a bitmask that indicates which columns are used in the query
			// If the last bit is set, it means that the rest of the columns are used
			if t.query.ColumnsUsed&(1<<i) == 0 && i < 62 {
				continue
			}

			dest = append(dest, &t.currentRow[i])
		}

		err = t.rows.Scan(dest...)
		if err != nil {
			return fmt.Errorf("error scanning the row: %v", err)
		}

	} else {
		t.currentRow = nil
	}

	return nil
}

func (t *ClickHouseCursor) Column(context *sqlite3.SQLiteContext, col int) error {
	if col < 0 || col >= len(t.currentRow) {
		context.ResultNull()
		return nil
	}

	if t.currentRow == nil {
		context.ResultNull()
	}

	if t.currentRow[col] == nil {
		context.ResultNull()
		return nil
	}

	switch val := t.currentRow[col].(type) {
	case nil:
		context.ResultNull()
	case *time.Time:
		context.ResultText(val.Format(time.RFC3339))
	case *timeMySQL:
		if val.Valid {
			if val.Time.IsZero() {
				context.ResultNull()
			} else if t.schema[col].Type == "DATE" {
				context.ResultText(val.Time.Format(time.DateOnly))
			} else {
				context.ResultText(val.Time.Format(time.RFC3339))
			}
		} else {
			context.ResultNull()
		}
	case *[]byte:
		if val == nil {
			context.ResultNull()
		} else {
			context.ResultBlob(*val)
		}
	case *bitMySQL:
		if val.Valid {
			context.ResultInt64(int64(val.Uint64))
		} else {
			context.ResultNull()
		}
	case *NullUint64:
		if val.Valid {
			context.ResultInt64(int64(val.Value)) // ClickHouse does not have unsigned integers, so we use int64
		} else {
			context.ResultNull()
		}
	case *sql.NullInt64:
		if val.Valid {
			context.ResultInt64(val.Int64)
		} else {
			context.ResultNull()
		}
	case *sql.NullFloat64:
		if val.Valid {
			context.ResultDouble(val.Float64)
		} else {
			context.ResultNull()
		}
	case *sql.NullString:
		if val.Valid {
			context.ResultText(val.String)
		} else {
			context.ResultNull()
		}
	default:
		// Try to convert the value to a JSON string
		jsonVal, err := json.Marshal(val)
		if err != nil {
			context.ResultNull()
		} else {
			context.ResultText(string(jsonVal))
		}

	}

	return nil
}

func (t *ClickHouseCursor) EOF() bool {
	return t.exhausted
}

func (t *ClickHouseCursor) Rowid() (int64, error) {
	return rand.Int64(), nil
}

func (t *ClickHouseCursor) Close() error {
	// Release the connection
	if t.connection != nil {
		// t.resetCursor() must always be called before closing the connection
		// because you can't release the connection with Close if the rows are not exhausted
		// or closed
		err := t.resetCursor()
		if err != nil {
			return fmt.Errorf("error resetting the cursor: %v", err)
		}
		err = t.connection.Close()
		if err != nil {
			return fmt.Errorf("error closing the connection: %v", err)
		}
		t.connection = nil
	}
	return nil
}
