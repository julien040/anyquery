package module

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"math/rand/v2"
	"strings"
	"sync"
	"time"

	"github.com/huandu/go-sqlbuilder"
	go_ora "github.com/sijms/go-ora/v2"

	"github.com/mattn/go-sqlite3"
)

type sDOPoint struct {
	X float64 `udt:"X"`
	Y float64 `udt:"Y"`
	Z float64 `udt:"Z"`
}
type sDOGeometry struct {
	GType    int64    `udt:"SDO_GTYPE`
	SRID     int64    `udt:"SDO_SRID"`
	Point    sDOPoint `udt:"SDO_POINT"`
	ElemInfo []int64  `udt:"SDO_GEOMETRYINFO"`
	Ordinate []int64  `udt:"SDO_ORDINATE"`
	//SDO_POINT SDO_POINT_TYPE,
	//SDO_ELEM_INFO SDO_ELEM_INFO_ARRAY,
	//SDO_ORDINATES SDO_ORDINATE_ARRAY);
}

// Fetch the schema of the table
// and which columns are primary keys
var fetchOracleSchemaSQLQuery = `
SELECT
	c.column_name,
	c.data_type,
	c.nullable,
	NVL(pk.position, 0) AS pk_position,
	CASE
		WHEN pk.position IS NOT NULL THEN 'PRIMARY KEY'
		ELSE 'NOT PRIMARY KEY'
	END AS column_role
FROM
	all_tab_columns c
	LEFT JOIN (
		SELECT
			acc.table_name,
			acc.owner,
			acc.column_name,
			acc.position
		FROM
			all_cons_columns acc
			JOIN all_constraints ac ON acc.constraint_name = ac.constraint_name
			AND acc.owner = ac.owner
		WHERE
			ac.constraint_type = 'P'
	) pk ON c.owner = pk.owner
	AND c.table_name = pk.table_name
	AND c.column_name = pk.column_name
WHERE
	c.table_name = :table_name
	AND c.owner = :schema_name
ORDER BY
	c.column_id
`

type OracleDBModule struct {
	pooler map[string]*sql.DB // A pooler for each connection string
	mtx    *sync.RWMutex      // To protect the pooler from concurrent access
}

// Retrieve or create a connection from the pooler of the pooler
func (m *OracleDBModule) GetDBConnection(connectionString string) (*sql.Conn, error) {
	m.mtx.RLock()
	if pool, ok := m.pooler[connectionString]; ok {
		m.mtx.RUnlock()
		return pool.Conn(context.Background())
	}
	m.mtx.RUnlock()

	// If the conn does not exist, create a new one
	db, err := sql.Open("oracle", connectionString)
	if err != nil {
		return nil, fmt.Errorf("error opening the database connection: %v", err)
	}

	// Thanks https://github.com/sijms/go-ora/issues/572
	err = go_ora.RegisterTypeWithOwner(db, "MDSYS", "number", "sdo_elem_info_array", nil)
	if err != nil {
		return nil, fmt.Errorf("can't register sdo_elem_info_array: %v", err)
	}
	err = go_ora.RegisterTypeWithOwner(db, "MDSYS", "number", "sdo_ordinate_array", nil)
	if err != nil {
		return nil, fmt.Errorf("can't register sdo_ordinate_array: %v", err)
	}
	err = go_ora.RegisterTypeWithOwner(db, "MDSYS", "SDO_POINT_TYPE", "", sDOPoint{})
	if err != nil {
		return nil, fmt.Errorf("can't register SDO_POINT_TYPE: %v", err)
	}
	err = go_ora.RegisterTypeWithOwner(db, "MDSYS", "SDO_GEOMETRY", "", sDOGeometry{})
	if err != nil {
		return nil, fmt.Errorf("can't register SDO_GEOMETRY: %v", err)
	}

	m.mtx.Lock()
	m.pooler[connectionString] = db
	m.mtx.Unlock()
	return db.Conn(context.Background())
}

type OracleDBTable struct {
	connection           *sql.Conn
	tableName            string
	schema               []databaseColumn
	module               *OracleDBModule
	connectionString     string
	supportsUpdateDelete bool
	primaryKeyColNames   []string
	transactionStarted   bool // Whether a BEGIN has been called
}

type OracleDBCursor struct {
	connection *sql.Conn
	tableName  string
	schema     []databaseColumn
	rows       *sql.Rows
	exhausted  bool
	currentRow []interface{}
	query      SQLQueryToExecute
}

func (m *OracleDBModule) Create(c *sqlite3.SQLiteConn, args []string) (sqlite3.VTab, error) {
	return m.Connect(c, args)
}

func (m *OracleDBModule) TransactionModule() {}

func (v *OracleDBModule) DestroyModule() {}

func (m *OracleDBModule) Connect(c *sqlite3.SQLiteConn, args []string) (sqlite3.VTab, error) {
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
	defer conn.Close()

	// Fetch the schema for the table
	rows, err := conn.QueryContext(context.Background(), fetchOracleSchemaSQLQuery, sql.Named("table_name", strings.ToUpper(table)), sql.Named("schema_name", strings.ToUpper(schemaName)))
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
		var columnName, dataType, nullable string
		var constraintType sql.NullString
		var pk_position int
		err = rows.Scan(&columnName, &dataType, &nullable, &pk_position, &constraintType)
		if err != nil {
			return nil, fmt.Errorf("error scanning the schema: %v", err)
		}
		columnType := "TEXT"
		typeSupported := false
		dataType = strings.ToLower(dataType)

		// This is an overview of the Oracle data types and their SQLite equivalents
		// https://docs.oracle.com/en/database/oracle/oracle-database/19/sqlrf/Data-Types.html
		//
		// If a type is missing, don't hesitate to open an issue on the GitHub repository
		switch {
		case strings.Contains(dataType, "char") || strings.HasSuffix(dataType, "rowid"):
			columnType = "TEXT"
			typeSupported = true
			defaultValue = ""
		case strings.Contains(dataType, "int") || strings.HasPrefix(dataType, "decimal") || strings.HasPrefix(dataType, "numeric") ||
			strings.HasPrefix(dataType, "dec"):
			columnType = "INTEGER"
			typeSupported = true
			defaultValue = 0
		case strings.Contains(dataType, "float") || strings.Contains(dataType, "double") || strings.Contains(dataType, "real"):
			columnType = "REAL"
			typeSupported = true
			defaultValue = 0.0
		case dataType == "date":
			columnType = "DATE"
			typeSupported = true
			defaultValue = "1970-01-01"
		case strings.HasPrefix(dataType, "timestamp"):
			columnType = "DATETIME"
			typeSupported = true
			defaultValue = "1970-01-01T00:00:00Z"

		case strings.HasPrefix(dataType, "interval"):
			columnType = "TEXT" // SQLite does not support intervals, so we store them as text
			typeSupported = false
			defaultValue = "" // No default value for intervals

		// Blob and binary data types
		case strings.Contains(dataType, "blob") || strings.Contains(dataType, "bfile") || strings.Contains(dataType, "raw") ||
			strings.Contains(dataType, "clob") || strings.Contains(dataType, "long"):
			columnType = "BLOB"
			typeSupported = true
			defaultValue = []byte{} // Default value for BLOBs is an empty byte slice

		case strings.Contains(dataType, "json"):
			columnType = "JSON"
			typeSupported = true
			defaultValue = "{}"
		case dataType == "xmltype":
			columnType = "TEXT"
			typeSupported = false
			defaultValue = ""

		case strings.Contains(dataType, "vector"):
			columnType = "JSON" // Treated as a JSON array
			typeSupported = true
			defaultValue = "[]"

		case strings.Contains(dataType, "any"):
			columnType = "UNKNOWN"
			typeSupported = false
			defaultValue = ""

		case strings.Contains(dataType, "geometry"):
			columnType = "UNKNOWN"
			typeSupported = false
			defaultValue = ""

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
			RemoteType:   dataType,
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
	return &OracleDBTable{
		tableName:            table,
		schema:               internalSchema,
		module:               m,
		connectionString:     connectionString,
		supportsUpdateDelete: supportsUpdateDelete,
		primaryKeyColNames:   primaryKeys,
	}, nil
}

func (t *OracleDBTable) Open() (sqlite3.VTabCursor, error) {
	// Create a new connection for each cursor
	conn, err := t.module.GetDBConnection(t.connectionString)
	if err != nil {
		return nil, fmt.Errorf("error getting a new connection: %v", err)
	}

	// Prepare the current row slice
	currentRow := make([]interface{}, len(t.schema))
	for i := range currentRow {
		switch t.schema[i].Type {
		case "INTEGER":
			currentRow[i] = new(sql.NullInt64)
		case "REAL":
			currentRow[i] = new(sql.NullFloat64)
		case "TEXT":
			currentRow[i] = new(sql.NullString)
		case "JSON":
			currentRow[i] = new([]byte)
		case "DATE", "DATETIME":
			currentRow[i] = new(sql.NullTime)
		case "BLOB":
			switch t.schema[i].RemoteType {
			case "blob":
				currentRow[i] = new(go_ora.Blob)
			case "bfile":
				currentRow[i] = new(go_ora.BFile)
			case "raw", "long raw", "long":
				currentRow[i] = new([]byte)
			case "clob":
				currentRow[i] = new(go_ora.Clob)
			}
		case "UNKNOWN":
			if t.schema[i].RemoteType == "geometry" {
				currentRow[i] = new(sDOGeometry) // Custom type for Oracle geometry
			} else {

				currentRow[i] = new(sql.NullString)
			}
		}
	}

	return &OracleDBCursor{
		connection: conn,
		tableName:  t.tableName,
		schema:     t.schema,
		currentRow: make([]interface{}, len(t.schema)),
	}, nil
}

// Check if the table supports partial updates
//
// Partial updates are updates that only update the columns that are provided
// Other columns in the Update call are replaced with a nil value
func (t *OracleDBTable) PartialUpdate() bool {
	return true
}

func (t *OracleDBTable) Disconnect() error {
	return t.Destroy()
}

func (t *OracleDBTable) releaseConnection() {
	if t.connection != nil {
		t.connection.Close()
		t.connection = nil
	}
}

func (t *OracleDBTable) Destroy() error {
	// Release the connection
	t.releaseConnection()
	return nil
}

var oracleQueryPlan = `
SELECT MAX(cost)
FROM plan_table
WHERE statement_id = '%d' 
`

// A function that will be called several times to check the best way to access the data.
// This function is called with different constraints and order by clauses
//
// To find the method, we will ask the database to explain the query and return the best method
func (t *OracleDBTable) BestIndex(cst []sqlite3.InfoConstraint, ob []sqlite3.InfoOrderBy, info sqlite3.IndexInformation) (*sqlite3.IndexResult, error) {
	// Create the SQL query
	queryBuilder, limitCstIndex, offsetCstIndex, used := efficientConstructSQLQuery(cst, ob, t.schema, t.tableName, info.ColUsed)
	queryBuilder.SetFlavor(sqlbuilder.Oracle)
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

	// We use the EXPLAIN command to get the query cost
	// This let us find the best way to access the data for SQLite

	statementID := rand.Int64()

	explainQuery := fmt.Sprintf("EXPLAIN PLAN SET STATEMENT_ID = '%d' FOR %s", statementID, rawQuery)
	_, err := t.connection.ExecContext(context.Background(), explainQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("error executing the explain query: %v", err)
	}

	var (
		cost    float64
		sqlCost sql.NullFloat64
	)

	// Fetch the cost of the query
	err = t.connection.QueryRowContext(context.Background(), fmt.Sprintf(oracleQueryPlan, statementID)).Scan(&sqlCost)
	if err != nil {
		return nil, fmt.Errorf("error fetching the query cost: %v", err)
	} else if sqlCost.Valid {
		cost = sqlCost.Float64
	} else {
		return nil, fmt.Errorf("the query cost for calculating the best querying plan is null")

	}

	// Reduce the cost if the query has a limit or offset
	if limitCstIndex != -1 {
		cost -= 0.1
	}
	if offsetCstIndex != -1 {
		cost -= 0.1
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

	// Set alreadyOrdered flag if the requested ordered columns are all supported
	alreadyOrdered := true
	for _, o := range ob {
		if o.Column < 0 || o.Column >= len(t.schema) {
			alreadyOrdered = false
			break
		}
		alreadyOrdered = alreadyOrdered && t.schema[o.Column].Supported
	}

	return &sqlite3.IndexResult{
		Used:           used,
		IdxStr:         string(serializedQuery),
		EstimatedCost:  cost,
		AlreadyOrdered: alreadyOrdered,
	}, nil
}

// Transactions related functions
func (t *OracleDBTable) Begin() error {
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

func (t *OracleDBTable) Commit() error {
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

func (t *OracleDBTable) Rollback() error {
	t.connection.ExecContext(context.Background(), "ROLLBACK")
	// Release the connection
	t.releaseConnection()
	t.transactionStarted = false
	return nil
}

// DML related functions
func (t *OracleDBTable) Insert(id any, vals []any) (int64, error) {
	rewriteArgs(&vals, t.schema)
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
	builder.SetFlavor(sqlbuilder.Oracle)

	query, args := builder.Build()
	query += sqlQuerySuffix

	// Rewrite the arguments to match the schema
	_, err := t.connection.ExecContext(context.Background(), query, args...)
	if err != nil {
		return 0, fmt.Errorf("error executing the insert query: %v", err)
	}

	return rand.Int64(), nil
}

func (t *OracleDBTable) Update(id any, vals []any) error {
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
	builder.SetFlavor(sqlbuilder.Oracle)

	query, args := builder.Build()
	query += sqlQuerySuffix

	_, err := t.connection.ExecContext(context.Background(), query, args...)
	if err != nil {
		return fmt.Errorf("error executing the update query: %v", err)
	}

	return nil
}

func (t *OracleDBTable) Delete(id any) error {
	if !t.supportsUpdateDelete {
		return fmt.Errorf("tables that support delete have one and only one primary key")
	}
	builder := sqlbuilder.NewDeleteBuilder()
	builder.DeleteFrom(t.tableName)
	builder.Where(builder.Equal(t.primaryKeyColNames[0], id))
	builder.SetFlavor(sqlbuilder.Oracle)

	query, args := builder.Build()
	query += sqlQuerySuffix

	_, err := t.connection.ExecContext(context.Background(), query, args...)
	if err != nil {
		return fmt.Errorf("error executing the delete query: %v", err)
	}

	return nil
}

func (t *OracleDBCursor) resetCursor() error {
	if t.rows != nil {
		t.rows.Close()
		t.rows = nil
	}

	return nil
}

func (t *OracleDBCursor) Filter(idxNum int, idxStr string, vals []interface{}) error {
	fmt.Println("Filter called")
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
	if offset != -1 {
		query.Query += " OFFSET " + fmt.Sprintf("%d", offset) + " ROWS"
	}
	if limit != -1 {
		query.Query += " FETCH NEXT " + fmt.Sprintf("%d", limit) + " ROWS ONLY"
	}

	// Request a connection if not already done

	// Execute the query
	fmt.Println("Executing query:", query.Query, "with params:", queryParams)
	rows, err := t.connection.QueryContext(context.Background(), query.Query, queryParams...)
	if err != nil {
		return fmt.Errorf("error executing the query: %v", err)
	}

	t.rows = rows
	return t.Next()
}

func (t *OracleDBCursor) Next() error {
	fmt.Println("Next() called")
	if t.rows == nil {
		return fmt.Errorf("no rows to iterate over")
	}
	hasMoreRows := t.rows.Next()
	t.exhausted = !hasMoreRows
	if t.rows.Err() != nil {
		return fmt.Errorf("error iterating over the rows: %v", t.rows.Err())
	}

	if hasMoreRows {
		// We'll scan the row into the currentRow slice

		// Scan the row into the currentRow slice
		dest := make([]interface{}, 0, len(t.schema))
		for i := range t.schema {
			// ColumnsUsed is a bitmask that indicates which columns are used in the query
			// If the last bit is set, it means that the rest of the columns are used

			// Continue if the column is not used in the query
			if t.query.ColumnsUsed&(1<<i) == 0 && i < 62 {
				continue
			}

			dest = append(dest, &t.currentRow[i])
		}

		fmt.Println("Scanning row with destination:")
		for i, d := range dest {
			fmt.Printf("  Column %d: %T\n", i, d)
		}

		err := t.rows.Scan(dest...)
		if err != nil {
			return fmt.Errorf("error scanning the row: %v", err)
		}
		fmt.Println("Row scanned successfully")
		for i, val := range t.currentRow {
			fmt.Printf("  Column %d: %v (%T)\n", i, val, val)
		}

	} else {
		t.currentRow = nil
	}

	return nil
}

func (t *OracleDBCursor) Column(context *sqlite3.SQLiteContext, col int) error {
	fmt.Printf("Column %d ", col)
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

	// Should not happen, but just in case
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

	case *sql.NullString:
		if val.Valid {
			context.ResultText(val.String)
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

	case *sql.NullTime:
		if val.Valid {
			context.ResultText(val.Time.Format(time.RFC3339))
		} else {
			context.ResultNull()
		}
	case time.Time:
		if !val.IsZero() {
			context.ResultText(val.Format(time.RFC3339))
		} else {
			context.ResultNull()
		}

	case *go_ora.Blob:
		if val.Valid {
			context.ResultBlob(val.Data)
		} else {
			context.ResultNull()
		}

	case *go_ora.BFile:
		if val.Valid {
			data, err := val.Read()
			if err != nil {
				return fmt.Errorf("error reading BFile: %v", err)
			}
			context.ResultBlob(data)
		} else {
			context.ResultNull()
		}

	case *go_ora.Clob:
		if val.Valid {
			context.ResultText(val.String)
		} else {
			context.ResultNull()
		}

	case []byte:
		if val == nil {
			context.ResultNull()
		} else {
			// Check if JSON
			if val[0] == '{' || val[0] == '[' {
				// Convert to a string
				context.ResultText(string(val))
			} else {
				context.ResultBlob(val)
			}
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

func (t *OracleDBCursor) EOF() bool {
	fmt.Println("EOF() called", t.exhausted)
	return t.exhausted
}

func (t *OracleDBCursor) Rowid() (int64, error) {
	return rand.Int64(), nil
}

func (t *OracleDBCursor) Close() error {
	fmt.Println("Close() called")
	// Release the connection
	err := t.resetCursor()
	if err != nil {
		return fmt.Errorf("error resetting the cursor: %v", err)
	}
	if t.connection != nil {
		t.connection.Close()
		t.connection = nil
	}
	return nil
}
