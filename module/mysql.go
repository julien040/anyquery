package module

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"math/rand/v2"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/huandu/go-sqlbuilder"
	"github.com/twpayne/go-geom/encoding/wkb"
	"github.com/twpayne/go-geom/encoding/wkt"

	mysql "github.com/go-sql-driver/mysql"
	"github.com/mattn/go-sqlite3"
)

var mysqlSuffix = "/* Query sent by Anyquery */"

// Fetch the schema of the table
// and which columns are primary keys
var fetchMySQLSchemaSQLQuery = `
SELECT DISTINCT
	C.table_schema,
	C.table_name,
	C.column_name,
	C.data_type,
	J.constraint_type,
	C.ordinal_position
FROM
	information_schema.columns AS C
	LEFT JOIN (
		SELECT
			column_name,
			constraint_type
		FROM
			information_schema.key_column_usage AS K
			JOIN information_schema.TABLE_CONSTRAINTS AS T ON K.constraint_name = T.constraint_name
		WHERE
			lower(K.table_schema) = lower(?)
			AND lower(K.table_name) = lower(?)
			AND constraint_type = 'PRIMARY KEY') 
	AS J ON C.column_name = J.column_name
WHERE
	lower(C.table_name) = lower(?)
	AND lower(C.table_schema) = lower(?)
ORDER BY
	C.ordinal_position;
`

// Types that corresponds to geometry, and must be converted to text using ST_AsText and ST_GeomFromText
var well_known_text_types = map[string]struct{}{
	"geometry":           {},
	"point":              {},
	"linestring":         {},
	"polygon":            {},
	"multipoint":         {},
	"multilinestring":    {},
	"multipolygon":       {},
	"geometrycollection": {},
	"geomcollection":     {},
}

type wellKnowTextGeo struct {
	StrRepresentation string
}

func (w *wellKnowTextGeo) Scan(src interface{}) error {
	if src == nil {
		return nil
	}
	switch v := src.(type) {
	case []byte:
		// https://dev.mysql.com/doc/refman/8.0/en/gis-data-formats.html
		// MySQL stores the geometry as a WKB (Well-Known Binary) format
		// prefixed by a 4-byte to indicate the SRID
		//
		// Therefore, we need to remove the first 4 bytes to get the WKB
		if len(v) < 4 {
			return fmt.Errorf("invalid WKB format. Missing the SRID: %v", v)
		}
		v = v[4:]

		// Unmarshal the byte array to geom.T
		geo, err := wkb.Unmarshal(v)
		if err != nil {
			return fmt.Errorf("error unmarshalling the geometry: %v", err)
		}
		// Marshal the geom.T to a well-known text
		wkt, err := wkt.Marshal(geo, wkt.EncodeOptionWithMaxDecimalDigits(8))
		if err != nil {
			return fmt.Errorf("error marshalling the geometry: %v", err)
		}
		w.StrRepresentation = wkt

	case string:
		w.StrRepresentation = v
	default:
		return fmt.Errorf("unsupported type for wellKnowTextGeo: %T", src)
	}
	return nil
}

type bitMySQL struct {
	Uint64 uint64
	Valid  bool
}

func (b *bitMySQL) Scan(src interface{}) error {
	if src == nil {
		b.Valid = false
		return nil
	}
	b.Valid = true
	switch v := src.(type) {
	case uint64:
		b.Uint64 = v
	case []byte:
		b.Uint64 = 0
		// Reverse the byte array and convert it to a uint64
		for i := len(v) - 1; i >= 0; i-- {
			b.Uint64 = b.Uint64<<8 | uint64(v[i])
		}
	default:
		return fmt.Errorf("unsupported type for bitMySQL: %T", src)
	}
	return nil
}

type timeMySQL struct {
	Time  time.Time
	Valid bool
}

func (t *timeMySQL) Scan(src interface{}) error {
	if src == nil {
		t.Valid = false
		return nil
	}
	t.Valid = true
	switch v := src.(type) {
	case time.Time:
		t.Time = v
	case string:
		t.Time, _ = time.Parse(time.RFC3339, v)
	case []byte:
		t.Time, _ = time.Parse(time.RFC3339, string(v))
	default:
		return fmt.Errorf("unsupported type for timeMySQL: %T", src)
	}
	return nil
}

type MySQLPlan struct {
	QueryBlock struct {
		SelectID int64 `json:"select_id"`
		CostInfo struct {
			QueryCost string `json:"query_cost"`
		} `json:"cost_info"`
	} `json:"query_block"`
}

type MySQLModule struct {
	pooler map[string]*sql.DB // A pooler for each connection string
	mtx    *sync.RWMutex      // To protect the pooler from concurrent access
}

// Retrieve or create a connection from the pooler of the pooler
func (m *MySQLModule) GetDBConnection(connectionString string) (*sql.Conn, error) {
	m.mtx.RLock()
	if pool, ok := m.pooler[connectionString]; ok {
		m.mtx.RUnlock()
		conn, err := pool.Conn(context.Background())
		return conn, err
	}
	m.mtx.RUnlock()

	pool, err := sql.Open("mysql", connectionString)
	if err != nil {
		return nil, fmt.Errorf("error connecting to the database: %v", err)
	}

	m.mtx.Lock()
	m.pooler[connectionString] = pool
	m.mtx.Unlock()
	return pool.Conn(context.Background())
}

type MySQLTable struct {
	connection           *sql.Conn
	tableName            string
	schema               []databaseColumn
	module               *MySQLModule
	connectionString     string
	supportsUpdateDelete bool
	primaryKeyColNames   []string
	transactionStarted   bool // Whether a BEGIN has been called
}

type MySQLCursor struct {
	connection   *sql.Conn
	tableName    string
	schema       []databaseColumn
	rows         *sql.Rows
	exhausted    bool
	currentRow   []interface{}
	rowsReturned int64
	limit        int64
}

func (m *MySQLModule) Create(c *sqlite3.SQLiteConn, args []string) (sqlite3.VTab, error) {
	return m.Connect(c, args)
}

func (m *MySQLModule) TransactionModule() {}

func (v *MySQLModule) DestroyModule() {}

func (m *MySQLModule) Connect(c *sqlite3.SQLiteConn, args []string) (sqlite3.VTab, error) {
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

	// Parse the tableName and split it into schema and table if needed
	schemaTable := strings.Split(table, ".")
	if len(schemaTable) > 1 {
		schemaName = schemaTable[0]
		table = schemaTable[1]

		// Remove any quotes, backticks or spaces around the schema and table names
		schemaName = strings.Trim(schemaName, "\" '`")
		table = strings.Trim(table, "\" '`")
	}

	// Rewrite the connection string to remove the protocol (not supported by the MySQL driver)
	connectionString = strings.TrimPrefix(connectionString, "mysql://")

	// Parse the connection string to get the database name
	parsed, err := mysql.ParseDSN(connectionString)
	if err != nil {
		return nil, fmt.Errorf("error parsing the connection string: %v", err)
	}
	parsed.ParseTime = true
	connectionString = parsed.FormatDSN()

	if schemaName == "" {
		schemaName = parsed.DBName
	}

	if schemaName == "" {
		return nil, fmt.Errorf("no database name found in the connection string")
	}
	// Open the database
	conn, err := m.GetDBConnection(connectionString)
	if err != nil {
		return nil, fmt.Errorf("error opening a connection to the database: %v", err)
	}

	// Return the connection
	defer conn.Close()

	// Fetch the schema for the table
	rows, err := conn.QueryContext(context.Background(), fetchMySQLSchemaSQLQuery, schemaName, table, table, schemaName)
	if err != nil {
		return nil, fmt.Errorf("error fetching the schema for the table: %v", err)
	}
	defer rows.Close()

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
		dataType = strings.ToLower(dataType)
		switch dataType {
		case "tinyint", "smallint", "mediumint", "int", "integer", "bigint", "unsigned big int", "year", "bit":
			columnType = "INTEGER"
			typeSupported = true
			defaultValue = 0
		case "decimal", "numeric", "real", "float", "double", "double precision":
			columnType = "REAL"
			typeSupported = true
			defaultValue = 0.0
		case "date", "datetime", "timestamp":
			columnType = "DATETIME"
			typeSupported = true
			defaultValue = time.Time{}.Format(time.RFC3339)
		case "char", "varchar", "text", "tinytext", "mediumtext", "longtext", "time": // Convert time to text
			columnType = "TEXT"
			typeSupported = true
			defaultValue = ""
		case "binary", "varbinary", "blob", "tinyblob", "mediumblob", "longblob", "bitea":
			columnType = "BLOB"
			typeSupported = true
			defaultValue = []byte{}
		case "enum", "set":
			columnType = "TEXT"
			typeSupported = true
			defaultValue = ""
		case "geometry", "point", "linestring", "polygon", "multipoint", "multilinestring", "multipolygon", "geometrycollection", "geomcollection":
			columnType = "TEXT"
			typeSupported = true
			defaultValue = ""
		case "json", "jsonb":
			columnType = "JSON"
			typeSupported = true
			defaultValue = ""
		case "boolean": // MySQL does not have a boolean type but just in case
			columnType = "INTEGER"
			typeSupported = true
			defaultValue = 0
		default:
			columnType = "UNKNOWN"
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
			RemoteType:   dataType,
			Supported:    typeSupported,
			DefaultValue: defaultValue,
		})
	}

	if rows.Err() != nil {
		return nil, fmt.Errorf("error iterating over the rows: %v", rows.Err())
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

	// Merge the schema with the table name
	table = fmt.Sprintf("`%s`.`%s`", schemaName, table)

	// Return the table instance
	return &MySQLTable{
		tableName:            table,
		schema:               internalSchema,
		module:               m,
		connectionString:     connectionString,
		supportsUpdateDelete: supportsUpdateDelete,
		primaryKeyColNames:   primaryKeys,
	}, nil
}

func (t *MySQLTable) Open() (sqlite3.VTabCursor, error) {
	// Create a new connection for each cursor
	conn, err := t.module.GetDBConnection(t.connectionString)
	if err != nil {
		return nil, fmt.Errorf("error getting a new connection: %v", err)
	}
	return &MySQLCursor{
		connection: conn,
		tableName:  t.tableName,
		schema:     t.schema,
		limit:      -1,
	}, nil
}

// Check if the table supports partial updates
//
// Partial updates are updates that only update the columns that are provided
// Other columns in the Update call are replaced with a nil value
func (t *MySQLTable) PartialUpdate() bool {
	return true
}

func (t *MySQLTable) Disconnect() error {
	return t.Destroy()
}

func (t *MySQLTable) releaseConnection() {
	if t.connection != nil {
		t.connection.Close()
		t.connection = nil
	}
}

func (t *MySQLTable) Destroy() error {
	// Release the connection
	t.releaseConnection()
	return nil
}

// A function that will be called several times to check the best way to access the data.
// This function is called with different constraints and order by clauses
//
// To find the method, we will ask the database to explain the query and return the best method
func (t *MySQLTable) BestIndex(cst []sqlite3.InfoConstraint, ob []sqlite3.InfoOrderBy, info sqlite3.IndexInformation) (*sqlite3.IndexResult, error) {
	// Create the SQL query
	queryBuilder, limitCstIndex, offsetCstIndex, used := constructSQLQuery(cst, ob, t.schema, t.tableName, sqlbuilder.MySQL)
	queryBuilder.SetFlavor(sqlbuilder.MySQL)
	rawQuery, args := queryBuilder.Build()
	rawQuery += sqlQuerySuffix

	// Request a connection
	if t.connection == nil {
		conn, err := t.module.GetDBConnection(t.connectionString)
		if err != nil {
			return nil, fmt.Errorf("error getting a new connection for best index: %v", err)
		}
		t.connection = conn
	}
	defer t.releaseConnection()

	// Explain the query and ignore any error if the query is not valid
	// This might happen with MySQL-wire compatible databases that don't use the same syntax
	minimumCost := 1e9
	explainQuery := "EXPLAIN (FORMAT JSON) " + rawQuery
	rows, _ := t.connection.QueryContext(context.Background(), explainQuery, args...)

	// Parse the result
	var plan MySQLPlan
	if rows != nil {
		for rows.Next() {
			var planRow string
			err := rows.Scan(&planRow)
			if err != nil {
				// Go to the next row
				continue
			}
			err = json.Unmarshal([]byte(planRow), &plan)
			if err == nil {
				// Get the cost of the query
				cost, err := strconv.ParseFloat(plan.QueryBlock.CostInfo.QueryCost, 64)
				if err == nil && cost < minimumCost {
					minimumCost = cost
				}
			}
		}
	}

	// Reduce the cost if the query has a limit or offset
	if limitCstIndex != -1 {
		minimumCost -= 0.1
	}
	if offsetCstIndex != -1 {
		minimumCost -= 0.1
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
		EstimatedRows:  25, // Default value for EstimatedRows
		AlreadyOrdered: alreadyOrdered,
	}, nil
}

// Transactions related functions
func (t *MySQLTable) Begin() error {
	// Acquire a connection. Every call to INSERT/UPDATE/DELETE will be preceded by a call to Begin
	var err error
	t.connection, err = t.module.GetDBConnection(t.connectionString)
	if err != nil {
		return fmt.Errorf("error getting a new connection for begin: %v", err)
	}
	t.connection.ExecContext(context.Background(), "BEGIN")
	t.transactionStarted = true

	return nil
}

func (t *MySQLTable) Commit() error {
	// If no transaction has been started, we don't send a COMMIT
	if !t.transactionStarted {
		return nil
	}
	t.connection.ExecContext(context.Background(), "COMMIT")
	// Release the connection so that it can be reused
	t.releaseConnection()
	t.transactionStarted = false
	return nil
}

func (t *MySQLTable) Rollback() error {
	t.connection.ExecContext(context.Background(), "ROLLBACK")
	// Release the connection
	t.releaseConnection()
	t.transactionStarted = false
	return nil
}

// DML related functions
func (t *MySQLTable) Insert(id any, vals []any) (int64, error) {
	builder := sqlbuilder.NewInsertBuilder()
	builder.InsertInto(t.tableName)
	cols := []string{}
	values := []interface{}{}
	for i, v := range vals {
		if v == nil {
			continue
		}
		cols = append(cols, t.schema[i].Realname)

		// Special case for geometry types
		value := v
		if _, ok := well_known_text_types[t.schema[i].RemoteType]; ok {
			value = sqlbuilder.Raw(fmt.Sprintf("ST_GeomFromText('%s')", v))
		}

		values = append(values, value)
	}
	builder.Cols(cols...)
	builder.Values(values...)
	builder.SetFlavor(sqlbuilder.MySQL)

	query, args := builder.Build()
	query += sqlQuerySuffix

	_, err := t.connection.ExecContext(context.Background(), query, args...)
	if err != nil {
		return 0, fmt.Errorf("error executing the insert query: %v", err)
	}

	return rand.Int64(), nil
}

func (t *MySQLTable) Update(id any, vals []any) error {
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

		value := v

		// Special case for geometry types
		// We use the ST_GeomFromText function to convert the text to a geometry type
		if _, ok := well_known_text_types[t.schema[i].RemoteType]; ok {
			value = sqlbuilder.Raw(fmt.Sprintf("ST_GeomFromText('%s')", v))
		}

		sets = append(sets, builder.Assign(t.schema[i].Realname, value))
	}

	builder.Set(sets...)
	builder.Where(builder.Equal(t.primaryKeyColNames[0], id))
	builder.SetFlavor(sqlbuilder.MySQL)

	query, args := builder.Build()
	query += sqlQuerySuffix

	_, err := t.connection.ExecContext(context.Background(), query, args...)
	if err != nil {
		return fmt.Errorf("error executing the update query: %v", err)
	}

	return nil
}

func (t *MySQLTable) Delete(id any) error {
	if !t.supportsUpdateDelete {
		return fmt.Errorf("tables that support delete have one and only one primary key")
	}
	builder := sqlbuilder.NewDeleteBuilder()
	builder.DeleteFrom(t.tableName)
	builder.Where(builder.Equal(t.primaryKeyColNames[0], id))
	builder.SetFlavor(sqlbuilder.MySQL)

	query, args := builder.Build()
	query += sqlQuerySuffix

	_, err := t.connection.ExecContext(context.Background(), query, args...)
	if err != nil {
		return fmt.Errorf("error executing the delete query: %v", err)
	}

	return nil
}

func (t *MySQLCursor) resetCursor() error {
	if t.rows != nil {
		t.rows.Close()
		t.rows = nil
	}
	t.limit = -1
	t.rowsReturned = 0
	t.exhausted = false
	t.currentRow = nil

	return nil
}

func (t *MySQLCursor) Filter(idxNum int, idxStr string, vals []interface{}) error {
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

func (t *MySQLCursor) Next() error {
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
		// Init an array of the same size as the number of columns
		values := make([]interface{}, len(t.schema))
		for i := range values {
			values[i] = new(interface{})
			switch t.schema[i].Type {
			case "INTEGER":
				switch t.schema[i].RemoteType {
				case "bit":
					values[i] = new(bitMySQL)
				default:
					values[i] = new(sql.NullInt64)
				}
			case "REAL":
				values[i] = new(sql.NullFloat64)
			case "TEXT":
				// If the column is a geometry type, we need to convert it to text
				// We do that using github.com/twpayne/go-geom/encoding/wkb
				if _, ok := well_known_text_types[t.schema[i].RemoteType]; ok {
					values[i] = new(wellKnowTextGeo)
				} else {
					values[i] = new(sql.NullString)
				}
			case "BLOB":
				values[i] = new([]byte)
			case "DATETIME":
				values[i] = new(timeMySQL)
			default:
				values[i] = new(interface{})
			}
		}
		err = t.rows.Scan(values...)
		if err != nil {
			return fmt.Errorf("error scanning the row: %v", err)
		}
		t.currentRow = make([]interface{}, len(values))
		for i, v := range values {
			if v == nil {
				t.currentRow[i] = nil
				continue
			}
			t.currentRow[i] = v
		}

	} else {
		t.currentRow = nil
	}

	return nil
}

func (t *MySQLCursor) Column(context *sqlite3.SQLiteContext, col int) error {
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
			context.ResultText(val.Time.Format(time.RFC3339))
		} else {
			context.ResultNull()
		}
	case *[]byte:
		if val == nil {
			context.ResultNull()
		} else {
			context.ResultBlob(*val)
		}
	case [16]byte: // UUID
		// Convert the UUID to a string
		context.ResultText(fmt.Sprintf("%x-%x-%x-%x-%x", val[0:4], val[4:6], val[6:8], val[8:10], val[10:]))
	case *bitMySQL:
		if val.Valid {
			context.ResultInt64(int64(val.Uint64))
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
	// Convert the geometry types to text (WKT)
	// using github.com/twpayne/go-geom/encoding/wkt
	case *wellKnowTextGeo:
		if val == nil {
			context.ResultNull()
		} else {
			context.ResultText(val.StrRepresentation)
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

func (t *MySQLCursor) EOF() bool {
	return t.exhausted
}

func (t *MySQLCursor) Rowid() (int64, error) {
	return rand.Int64(), nil
}

func (t *MySQLCursor) Close() error {
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
