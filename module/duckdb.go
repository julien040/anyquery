package module

import (
	"encoding/json"
	"fmt"
	"math/rand/v2"
	"strings"

	"github.com/huandu/go-sqlbuilder"
	"github.com/julien040/anyquery/other/duckdb"

	"github.com/mattn/go-sqlite3"
)

var fetchDuckDBSchemaSQLQuery = `
SELECT DISTINCT
	lower(column_name) as column_name,
	lower(data_type) as data_type,
FROM
	information_schema.columns
WHERE lower(table_schema) = lower('%s')
AND lower(table_name) = lower('%s')
ORDER BY
	ordinal_position ASC;
`

type DuckDBModule struct {
}

type DuckDBTable struct {
	tableName        string
	schema           []databaseColumn
	connectionString string
}

type DuckDBCursor struct {
	tableName        string
	schema           []databaseColumn
	exhausted        bool
	currentRow       map[string]interface{}
	rowsReturned     int64
	limit            int64
	connectionString string

	rows   <-chan map[string]interface{}
	rowErr <-chan error
}

func (m *DuckDBModule) Create(c *sqlite3.SQLiteConn, args []string) (sqlite3.VTab, error) {
	return m.Connect(c, args)
}

func (m *DuckDBModule) TransactionModule() {}

func (v *DuckDBModule) DestroyModule() {}

func (m *DuckDBModule) Connect(c *sqlite3.SQLiteConn, args []string) (sqlite3.VTab, error) {

	// Fetch the arguments
	connectionString := ""
	table := ""
	schemaName := "main"
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

	// Fetch the schema for the table
	rows, errChan := duckdb.RunDuckDBQuery(connectionString, fmt.Sprintf(fetchDuckDBSchemaSQLQuery, schemaName, table))
	if len(errChan) > 0 {
		rowErr := <-errChan
		if rowErr != nil {
			return nil, fmt.Errorf("error fetching the schema for the table: %v", rowErr)
		}
	}

	// Iterate over the rows and create the schema
	internalSchema := []databaseColumn{}
	schema := strings.Builder{}
	schema.WriteString("CREATE TABLE x(\n")
	firstRow := true
	for row := range rows {
		if len(errChan) > 0 {
			err := <-errChan
			if err != nil {
				return nil, fmt.Errorf("error fetching the schema for the table: %v", err)
			}
		}

		var dataType, columnName string
		if val, ok := row["data_type"]; ok {
			if str, ok := val.(string); ok {
				dataType = strings.ToLower(str)
			} else {
				return nil, fmt.Errorf("invalid data type value: %v", val)
			}
		} else {
			return nil, fmt.Errorf("missing data_type in the row: %v", row)
		}

		if val, ok := row["column_name"]; ok {
			if str, ok := val.(string); ok {
				columnName = strings.ToLower(str)
			} else {
				return nil, fmt.Errorf("invalid column name value: %v", val)
			}
		} else {
			return nil, fmt.Errorf("missing column_name in the row: %v", row)
		}

		// Map DuckDB types to SQLite types
		columnType := "TEXT"  // Default to TEXT for unknown types
		typeSupported := true // Assume all types are supported by default

		switch {
		// Complex/Nested types
		// This case must be at the top because for example, a map is written as "MAP(VARCHAR, INTEGER)"
		// That would first match VARCHAR, and not map
		case strings.HasPrefix(dataType, "array"), strings.HasPrefix(dataType, "list"),
			strings.HasPrefix(dataType, "map"), strings.HasPrefix(dataType, "struct"), dataType == "json":
			columnType = "JSON"   // Store as JSON text
			typeSupported = false // These types are too different for SQLite. Any WHERE clause will not work as expected
		// Union types are treated as unknown in SQLite
		case strings.HasPrefix(dataType, "union"):
			columnType = "UNKNOWN"
			typeSupported = false
		// Integer types
		case strings.HasPrefix(dataType, "tinyint"), strings.HasPrefix(dataType, "smallint"),
			strings.HasPrefix(dataType, "int"), strings.HasPrefix(dataType, "bigint"),
			strings.HasPrefix(dataType, "hugeint"), strings.HasPrefix(dataType, "utinyint"),
			strings.HasPrefix(dataType, "usmallint"), strings.HasPrefix(dataType, "uint"),
			strings.HasPrefix(dataType, "ubigint"), strings.HasPrefix(dataType, "uhugeint"):
			columnType = "INTEGER"

		// Floating point types
		case strings.HasPrefix(dataType, "float"), strings.HasPrefix(dataType, "double"),
			dataType == "real", strings.HasPrefix(dataType, "decimal"),
			strings.HasPrefix(dataType, "numeric"):
			columnType = "REAL"

		// Boolean type
		case dataType == "bool", dataType == "boolean", dataType == "logical":
			columnType = "INTEGER" // SQLite represents boolean as 0/1

		// Text/String types
		case strings.Contains(dataType, "char"), strings.Contains(dataType, "text"), strings.Contains(dataType, "enum"),
			dataType == "string", dataType == "varchar", dataType == "uuid":
			columnType = "TEXT"

		// Binary types
		case strings.Contains(dataType, "blob"), strings.Contains(dataType, "binary"),
			dataType == "bytea", dataType == "varbinary":
			columnType = "BLOB"

		// Bit/Bitstring types
		case dataType == "bit", dataType == "bitstring":
			columnType = "TEXT" // Represent as text in SQLite

		// Date/Time types
		case dataType == "date", strings.HasPrefix(dataType, "time"),
			dataType == "datetime", dataType == "timestamptz",
			strings.HasPrefix(dataType, "timestamp"), dataType == "interval":
			columnType = "TEXT" // Store as ISO format text

		// Unknown or unsupported types
		default:
			columnType = "TEXT" // Default fallback
			typeSupported = false
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

	schema.WriteString("\n)")

	// Check if any errors were encountered while fetching the schema
	if len(errChan) > 0 {
		err := <-errChan
		if err != nil {
			return nil, fmt.Errorf("error fetching the schema for the table: %v", err)
		}
	}

	if len(internalSchema) == 0 {
		return nil, fmt.Errorf("no columns found for the table. Make sure the table exists and has columns")
	}

	// Declare the virtual table
	err := c.DeclareVTab(schema.String())
	if err != nil {
		return nil, fmt.Errorf("error declaring the virtual table: %v", err)
	}

	// Merge the schema with the table name
	table = fmt.Sprintf("\"%s\".\"%s\"", schemaName, table)

	// Return the table instance
	return &DuckDBTable{
		tableName:        table,
		schema:           internalSchema,
		connectionString: connectionString,
	}, nil
}

func (t *DuckDBTable) Open() (sqlite3.VTabCursor, error) {
	return &DuckDBCursor{
		tableName:        t.tableName,
		schema:           t.schema,
		limit:            -1,
		connectionString: t.connectionString,
	}, nil
}

func (t *DuckDBTable) Disconnect() error {
	return t.Destroy()
}

func (t *DuckDBTable) Destroy() error {
	return nil
}

// A function that will be called several times to check the best way to access the data.
// This function is called with different constraints and order by clauses
//
// To find the method, we will ask the database to explain the query and return the best method
func (t *DuckDBTable) BestIndex(cst []sqlite3.InfoConstraint, ob []sqlite3.InfoOrderBy, info sqlite3.IndexInformation) (*sqlite3.IndexResult, error) {
	// Create the SQL query
	queryBuilder, limitCstIndex, offsetCstIndex, used := efficientConstructSQLQuery(cst, ob, t.schema, t.tableName, info.ColUsed, sqlbuilder.PostgreSQL)
	queryBuilder.SetFlavor(sqlbuilder.PostgreSQL)
	rawQuery, args := queryBuilder.Build()

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
		EstimatedRows:  25,
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
func (t *DuckDBTable) Insert(id any, vals []any) (int64, error) {
	return 0, fmt.Errorf("insert operation is not supported for DuckDB tables")
}

func (t *DuckDBTable) Update(id any, vals []any) error {
	return fmt.Errorf("update operation is not supported for DuckDB tables")
}

func (t *DuckDBTable) Delete(id any) error {
	return fmt.Errorf("delete operation is not supported for DuckDB tables")
}

func (t *DuckDBTable) PartialUpdate() bool {
	return false
}

func (t *DuckDBCursor) resetCursor() error {
	t.limit = -1
	t.rowsReturned = 0
	t.exhausted = false
	t.currentRow = nil

	return nil
}

func (t *DuckDBCursor) Filter(idxNum int, idxStr string, vals []interface{}) error {
	// Reset the cursor as Filter might be called multiple times
	t.resetCursor()

	// Reconstruct the query and its arguments
	var query SQLQueryToExecute
	err := json.Unmarshal([]byte(idxStr), &query)
	if err != nil {
		return fmt.Errorf("error unmarshalling the query: %v", err)
	}

	// Get the LIMIT AND OFFSET values
	// and remove them from the query so that we can pass these arguments to the query
	limit := int64(-1)
	offset := int64(-1)
	queryParams := []interface{}{}
	for i, c := range vals {
		switch i {
		case query.LimitIndex:
			limit = c.(int64)
		case query.OffsetIndex:
			offset = c.(int64)
		default:
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

	// Interpolate the arguments into the query
	interpolatedQuery, err := sqlbuilder.PostgreSQL.Interpolate(query.Query, queryParams)
	if err != nil {
		return fmt.Errorf("query too complex to interpolate: %v", err)
	}

	// Run the query
	rows, rowErr := duckdb.RunDuckDBQuery(t.connectionString, interpolatedQuery)
	if len(rowErr) > 0 {
		rowError := <-rowErr
		if rowError != nil {
			return fmt.Errorf("error running the query: %v", rowError)
		}
	}
	t.rows = rows
	t.rowErr = rowErr

	// Call Next to read the first row
	t.rowsReturned = 0
	t.exhausted = false
	t.currentRow = nil
	if err := t.Next(); err != nil {
		return fmt.Errorf("error reading the first row: %v", err)
	}

	return nil
}

func (t *DuckDBCursor) Next() error {
	t.rowsReturned++
	if t.rows == nil {
		return fmt.Errorf("no rows to iterate over")
	}

	select {
	case row, ok := <-t.rows:
		if !ok {
			t.exhausted = true
			t.currentRow = nil
			return nil // No more rows to read
		}
		t.currentRow = row
		if t.limit != -1 && t.rowsReturned >= t.limit {
			t.exhausted = true
			t.rows = nil // Clear the rows channel to indicate exhaustion
		}

	case err, ok := <-t.rowErr:
		if !ok {
			t.rowErr = nil  // Remove the rowErr channel to fetch all rows
			return t.Next() // Try to read the next row
		}
		if err != nil {
			t.exhausted = true
			t.currentRow = nil
			return fmt.Errorf("error reading the next row: %v", err)
		}
	}

	return nil
}

func (t *DuckDBCursor) Column(context *sqlite3.SQLiteContext, col int) error {
	if col < 0 {
		context.ResultNull()
		return nil
	}

	if t.currentRow == nil {
		context.ResultNull()
	}

	// Get the column name from the schema
	if col >= len(t.schema) {
		return fmt.Errorf("column index %d out of range for schema with %d columns", col, len(t.schema))
	}
	column := t.schema[col]
	val, ok := t.currentRow[column.Realname]
	if !ok {
		context.ResultNull()
		return nil // Column not found in the current row, return NULL
	}

	// ResolvedVal will be one of the type returned by encoding/json
	switch resolvedVal := val.(type) {
	case nil:
		context.ResultNull()
	case float64:
		// JSON numbers are float64, so we need to convert them to int64 or float64
		// following the type of the column
		switch column.Type {
		case "INTEGER":
			context.ResultInt64(int64(resolvedVal))
		case "REAL":
			context.ResultDouble(resolvedVal)
		default:
			context.ResultText(fmt.Sprintf("%f", resolvedVal))
		}
	case string:
		// If the column is TEXT, we return the string value
		if column.Type == "BLOB" {
			// If the column is BLOB, we return the string as a byte slice
			context.ResultBlob([]byte(resolvedVal))
		} else {
			context.ResultText(resolvedVal)
		}
	case bool:
		// If the column is BOOLEAN, we return 0 or 1
		if resolvedVal {
			context.ResultInt64(1)
		} else {
			context.ResultInt64(0)
		}
	case []byte:
		// If the column is BLOB, we return the byte slice
		context.ResultBlob(resolvedVal)
	default:
		// Convert the value to a JSON string if it is not a supported type
		jsonValue, err := json.Marshal(resolvedVal)
		if err != nil {
			return fmt.Errorf("error marshalling value to JSON: %v", err)
		}
		context.ResultText(string(jsonValue))
	}

	return nil
}

func (t *DuckDBCursor) EOF() bool {
	return t.exhausted
}

func (t *DuckDBCursor) Rowid() (int64, error) {
	return rand.Int64(), nil
}

func (t *DuckDBCursor) Close() error {
	return nil
}
