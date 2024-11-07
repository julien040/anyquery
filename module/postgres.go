package module

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"math/rand/v2"
	"net"
	"net/netip"
	"strings"
	"sync"
	"time"

	"github.com/huandu/go-sqlbuilder"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/mattn/go-sqlite3"
)

var postgresSuffix = "/* Query sent by Anyquery */"

// Fetch the schema of the table
// and which columns are primary keys
var fetchPGSchemaSQLQuery = `
SELECT
	C.table_schema,
	C.table_name,
	C.column_name,
	C.data_type,
	J.constraint_type,
	C.ordinal_position
FROM
	information_schema. "columns" C
	LEFT JOIN (
		SELECT
			column_name,
			constraint_type
		FROM
			information_schema.key_column_usage K
			JOIN information_schema.TABLE_CONSTRAINTS T ON K. "constraint_name" = T. "constraint_name"
		WHERE
			K.table_name = $1
			AND K.table_schema = $2
			AND constraint_type = 'PRIMARY KEY') J ON C. "column_name" = J. "column_name"
WHERE
	table_name = $1
	AND table_schema = $2;
`

type PostgresPlan []struct {
	Plan struct {
		StartupCost float64 `json:"Startup Cost"`
		TotalCost   float64 `json:"Total Cost"`
	}
}

type PostgresModule struct {
	pooler map[string]*pgxpool.Pool // A pooler for each connection string
	mtx    *sync.RWMutex            // To protect the pooler from concurrent access
}

// Retrieve or create a connection from the pooler of the pooler
func (m *PostgresModule) GetDBConnection(connectionString string) (*pgxpool.Conn, error) {
	m.mtx.RLock()
	if pool, ok := m.pooler[connectionString]; ok {
		m.mtx.RUnlock()
		conn, err := pool.Acquire(context.Background())
		return conn, err
	}
	m.mtx.RUnlock()

	pool, err := pgxpool.New(context.Background(), connectionString)
	if err != nil {
		return nil, fmt.Errorf("error connecting to the database: %v", err)
	}

	m.mtx.Lock()
	m.pooler[connectionString] = pool
	m.mtx.Unlock()
	return pool.Acquire(context.Background())
}

type PostgresTable struct {
	connection           *pgxpool.Conn
	tableName            string
	schema               []databaseColumn
	module               *PostgresModule
	connectionString     string
	supportsUpdateDelete bool
	primaryKeyColNames   []string
	transactionStarted   bool // Whether a BEGIN has been called
}

type PostgresCursor struct {
	connection *pgxpool.Conn
	tableName  string
	schema     []databaseColumn
	rows       pgx.Rows
	exhausted  bool
	currentRow []interface{}
}

func (m *PostgresModule) Create(c *sqlite3.SQLiteConn, args []string) (sqlite3.VTab, error) {
	return m.Connect(c, args)
}

func (m *PostgresModule) TransactionModule() {}

func (v *PostgresModule) DestroyModule() {}

func (m *PostgresModule) Connect(c *sqlite3.SQLiteConn, args []string) (sqlite3.VTab, error) {
	// Init the structure
	if m.pooler == nil {
		m.pooler = make(map[string]*pgxpool.Pool)
	}
	if m.mtx == nil {
		m.mtx = &sync.RWMutex{}
	}

	// Fetch the arguments
	connectionString := ""
	table := ""
	schemaName := "public"
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

	// Parse the tableName and split it into schema and table if needed
	schemaTable := strings.Split(table, ".")
	if len(schemaTable) > 1 {
		schemaName = schemaTable[0]
		table = schemaTable[1]

		// Remove any quotes, backticks or spaces around the schema and table names
		schemaName = strings.Trim(schemaName, "\" '`")
		table = strings.Trim(table, "\" '`")
	}

	// Open the database
	conn, err := m.GetDBConnection(connectionString)
	if err != nil {
		return nil, err
	}

	// Return the connection
	defer conn.Release()

	// Fetch the schema for the table
	rows, err := conn.Query(context.Background(), fetchPGSchemaSQLQuery, table, schemaName)
	if err != nil {
		return nil, fmt.Errorf("error fetching the schema for the table: %v", err)
	}

	// Replace table name by its representation
	table = fmt.Sprintf("\"%s\".\"%s\"", schemaName, table)

	// Iterate over the rows and create the schema
	internalSchema := []databaseColumn{}
	schema := strings.Builder{}
	primaryKeys := []string{}
	schema.WriteString("CREATE TABLE x(\n")
	firstRow := true
	var defaultValue interface{}
	for rows.Next() {
		var tableSchema, tableName, columnName, dataType string
		var constraintType sql.NullString
		var ordinal_position int
		err = rows.Scan(&tableSchema, &tableName, &columnName, &dataType, &constraintType, &ordinal_position)
		if err != nil {
			return nil, fmt.Errorf("error scanning the schema: %v", err)
		}
		columnType := "TEXT"
		typeSupported := false
		switch dataType {
		case "integer", "bigint", "smallint", "int", "int2", "int4", "int8",
			"serial", "bigserial", "smallserial", "serial2", "serial4", "serial8":
			columnType = "INTEGER"
			typeSupported = true
			defaultValue = 0
		case "real", "double precision", "numeric",
			"decimal", "money", "float", "float4", "float8":
			columnType = "REAL"
			typeSupported = true
			defaultValue = 0.0
		case "boolean":
			columnType = "BOOLEAN"
		case "date", "time", "timestamp", "timestamptz", "timetz", "interval":
			columnType = "DATETIME"
			typeSupported = true
			defaultValue = ""
		case "text", "character", "character varying", "varchar", "char", "string":
			columnType = "TEXT"
			typeSupported = true
			defaultValue = ""
		case "bytea":
			columnType = "BLOB"
			typeSupported = true
			defaultValue = []byte{}
		default:
			columnType = "TEXT"
			typeSupported = false // Fail safe
			defaultValue = ""
		}

		localColumnName := transformSQLiteValidName(columnName)
		if !firstRow {
			schema.WriteString(",\n")
		}
		firstRow = false

		if constraintType.Valid && constraintType.String == "PRIMARY KEY" {
			primaryKeys = append(primaryKeys, localColumnName)
		}

		schema.WriteString(fmt.Sprintf("  \"%s\" %s", localColumnName, columnType))
		internalSchema = append(internalSchema, databaseColumn{
			Realname:     columnName,
			SQLiteName:   localColumnName,
			Type:         columnType,
			Supported:    typeSupported,
			DefaultValue: defaultValue,
		})
	}
	if len(primaryKeys) > 0 {
		schema.WriteString(fmt.Sprintf(",\n  PRIMARY KEY (%s)", strings.Join(primaryKeys, ", ")))
	}

	supportsUpdateDelete := len(primaryKeys) == 1

	if len(primaryKeys) == 0 {
		schema.WriteString("\n)")
	} else {
		schema.WriteString("\n) WITHOUT ROWID;")
	}

	if len(internalSchema) == 0 {
		return nil, fmt.Errorf("no columns found for the table")
	}

	// Declare the virtual table
	err = c.DeclareVTab(schema.String())
	if err != nil {
		return nil, fmt.Errorf("error declaring the virtual table: %v", err)
	}

	// Return the table instance
	return &PostgresTable{
		tableName:            table,
		schema:               internalSchema,
		module:               m,
		connectionString:     connectionString,
		supportsUpdateDelete: supportsUpdateDelete,
		primaryKeyColNames:   primaryKeys,
	}, nil
}

func (t *PostgresTable) Open() (sqlite3.VTabCursor, error) {
	// Create a new connection for each cursor
	conn, err := t.module.GetDBConnection(t.connectionString)
	if err != nil {
		return nil, fmt.Errorf("error getting a new connection: %v", err)
	}
	return &PostgresCursor{
		connection: conn,
		tableName:  t.tableName,
		schema:     t.schema,
	}, nil
}

// Check if the table supports partial updates
//
// Partial updates are updates that only update the columns that are provided
// Other columns in the Update call are replaced with a nil value
func (t *PostgresTable) PartialUpdate() bool {
	return true
}

func (t *PostgresTable) Disconnect() error {
	return t.Destroy()
}

func (t *PostgresTable) releaseConnection() {
	if t.connection != nil {
		t.connection.Release()
		t.connection = nil
	}
}

func (t *PostgresTable) Destroy() error {
	// Release the connection
	t.releaseConnection()
	return nil
}

// A function that will be called several times to check the best way to access the data.
// This function is called with different constraints and order by clauses
//
// To find the method, we will ask the database to explain the query and return the best method
func (t *PostgresTable) BestIndex(cst []sqlite3.InfoConstraint, ob []sqlite3.InfoOrderBy) (*sqlite3.IndexResult, error) {
	// Create the SQL query
	queryBuilder, limitCstIndex, offsetCstIndex, used := constructSQLQuery(cst, ob, t.schema, t.tableName)
	queryBuilder.SetFlavor(sqlbuilder.PostgreSQL)
	rawQuery, args := queryBuilder.Build()
	rawQuery += postgresSuffix

	// Request a connection
	if t.connection == nil {
		conn, err := t.module.GetDBConnection(t.connectionString)
		if err != nil {
			return nil, fmt.Errorf("error getting a new connection for best index: %v", err)
		}
		t.connection = conn
	}
	defer t.releaseConnection()

	// Explain the query
	explainQuery := "EXPLAIN (FORMAT JSON) " + rawQuery
	rows, err := t.connection.Query(context.Background(), explainQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("error explaining the query to compute the query plan: %v", err)
	}

	// Parse the result
	var plan PostgresPlan
	for rows.Next() {
		var planRow string
		err = rows.Scan(&planRow)
		if err != nil {
			return nil, fmt.Errorf("error scanning the plan from pg: %v", err)
		}
		err = json.Unmarshal([]byte(planRow), &plan)
		if err != nil {
			return nil, fmt.Errorf("error unmarshalling the plan from pg: %v", err)
		}
	}

	// Find the best plan
	minimumCost := 1e9
	for _, p := range plan {
		minimumCost = min(minimumCost, p.Plan.TotalCost)
	}

	// Reduce the cost if the query has a limit or offset
	if limitCstIndex != -1 {
		minimumCost -= 0.1
	}
	if offsetCstIndex != -1 {
		minimumCost -= 0.1
	}

	query := &SQLQueryToExecute{
		Query:       rawQuery,
		Args:        args,
		LimitIndex:  limitCstIndex,
		OffsetIndex: offsetCstIndex,
	}

	// Serialize the query as a JSON object
	serializedQuery, err := json.Marshal(query)
	if err != nil {
		return nil, fmt.Errorf("error serializing the query: %v", err)
	}

	return &sqlite3.IndexResult{
		Used:           used,
		IdxStr:         string(serializedQuery),
		EstimatedCost:  minimumCost,
		EstimatedRows:  25,   // Default value for EstimatedRows
		AlreadyOrdered: true, // To avoid SQLite reordering the results and requesting all rows
	}, nil
}

// Transactions related functions
func (t *PostgresTable) Begin() error {
	// Acquire a connection. Every call to INSERT/UPDATE/DELETE will be preceded by a call to Begin
	var err error
	t.connection, err = t.module.GetDBConnection(t.connectionString)
	if err != nil {
		return fmt.Errorf("error getting a new connection for begin: %v", err)
	}
	t.connection.Exec(context.Background(), "BEGIN")
	t.transactionStarted = true

	return nil
}

func (t *PostgresTable) Commit() error {
	// If no transaction has been started, we don't send a COMMIT
	if !t.transactionStarted {
		return nil
	}
	t.connection.Exec(context.Background(), "COMMIT")
	// Release the connection so that it can be reused
	t.releaseConnection()
	t.transactionStarted = false
	return nil
}

func (t *PostgresTable) Rollback() error {
	t.connection.Exec(context.Background(), "ROLLBACK")
	// Release the connection
	t.releaseConnection()
	t.transactionStarted = false
	return nil
}

// DML related functions
func (t *PostgresTable) Insert(id any, vals []any) (int64, error) {
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
	builder.SetFlavor(sqlbuilder.PostgreSQL)

	query, args := builder.Build()
	query += postgresSuffix

	_, err := t.connection.Exec(context.Background(), query, args...)
	if err != nil {
		return 0, fmt.Errorf("error executing the insert query: %v", err)
	}

	return rand.Int64(), nil
}

func (t *PostgresTable) Update(id any, vals []any) error {
	if !t.supportsUpdateDelete {
		return fmt.Errorf("tables that support update have one and only one primary key")
	}
	builder := sqlbuilder.NewUpdateBuilder()
	builder.Update(t.tableName)
	sets := []string{}
	for i, v := range vals {
		if v == nil {
			continue
		}
		sets = append(sets, builder.Assign(t.schema[i].Realname, v))
	}

	builder.Set(sets...)
	builder.Where(builder.Equal(t.primaryKeyColNames[0], id))
	builder.SetFlavor(sqlbuilder.PostgreSQL)

	query, args := builder.Build()
	query += postgresSuffix

	_, err := t.connection.Exec(context.Background(), query, args...)
	if err != nil {
		return fmt.Errorf("error executing the update query: %v", err)
	}

	return nil
}

func (t *PostgresTable) Delete(id any) error {
	if !t.supportsUpdateDelete {
		return fmt.Errorf("tables that support delete have one and only one primary key")
	}
	builder := sqlbuilder.NewDeleteBuilder()
	builder.DeleteFrom(t.tableName)
	builder.Where(builder.Equal(t.primaryKeyColNames[0], id))
	builder.SetFlavor(sqlbuilder.PostgreSQL)

	query, args := builder.Build()
	query += postgresSuffix

	_, err := t.connection.Exec(context.Background(), query, args...)
	if err != nil {
		return fmt.Errorf("error executing the delete query: %v", err)
	}

	return nil
}

func (t *PostgresCursor) resetCursor() error {
	if t.rows != nil {
		t.rows.Close()
		t.rows = nil
	}

	return nil
}

func (t *PostgresCursor) Filter(idxNum int, idxStr string, vals []interface{}) error {
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
	}
	if offset != -1 {
		query.Query += fmt.Sprintf(" OFFSET %d", offset)
	}

	// Execute the query
	rows, err := t.connection.Query(context.Background(), query.Query, queryParams...)
	if err != nil {
		return fmt.Errorf("error executing the query: %v", err)
	}

	t.rows = rows
	return t.Next()
}

func (t *PostgresCursor) Next() error {
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
		t.currentRow, err = t.rows.Values()
		if err != nil {
			return fmt.Errorf("error getting the values of the row: %v", err)
		}
	} else {
		t.currentRow = nil
	}

	return nil
}

type Valuable interface {
	Value() (driver.Value, error)
}

func (t *PostgresCursor) Column(context *sqlite3.SQLiteContext, col int) error {
	if col < 0 || col >= len(t.currentRow) {
		context.ResultNull()
		return nil
	}

	if t.currentRow == nil {
		context.ResultNull()
	}

	switch val := t.currentRow[col].(type) {
	case nil:
		context.ResultNull()
	case uint8, uint16, uint32, uint64, int8, int16, int32, int64, int:
		context.ResultInt64(castInt(val))
	case float64, float32:
		context.ResultDouble(castFloat(val))
	case bool:
		if val {
			context.ResultInt64(1)
		} else {
			context.ResultInt64(0)
		}
	case string:
		context.ResultText(val)
	case time.Time:
		context.ResultText(val.Format(time.RFC3339))
	case []byte:
		context.ResultBlob(val)
	case net.Addr:
		context.ResultText(val.String())
	case netip.Prefix:
		context.ResultText(val.String())
	case net.HardwareAddr:
		context.ResultText(val.String())
	case [16]byte: // UUID
		// Convert the UUID to a string
		context.ResultText(fmt.Sprintf("%x-%x-%x-%x-%x", val[0:4], val[4:6], val[6:8], val[8:10], val[10:]))
	case Valuable:
		v, err := val.Value()
		if err != nil {
			context.ResultNull()
		}
		switch v := v.(type) {
		case int64:
			context.ResultInt64(v)
		case float64:
			context.ResultDouble(v)
		case bool:
			if v {
				context.ResultInt64(1)
			} else {
				context.ResultInt64(0)
			}
		case string:
			context.ResultText(v)
		case []byte:
			context.ResultBlob(v)
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

func (t *PostgresCursor) EOF() bool {
	return t.exhausted
}

func (t *PostgresCursor) Rowid() (int64, error) {
	return rand.Int64(), nil
}

func (t *PostgresCursor) Close() error {
	// Release the connection
	if t.connection != nil {
		t.connection.Release()
		t.connection = nil
	}
	return t.resetCursor()
}
