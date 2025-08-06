package module

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"math/big"
	"math/rand/v2"
	"net"
	"net/url"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/huandu/go-sqlbuilder"
	"gopkg.in/inf.v0"

	gocql "github.com/apache/cassandra-gocql-driver/v2"
	"github.com/mattn/go-sqlite3"
)

var fetchCassandraSchemaSQLQuery = `
SELECT
	column_name,
	kind,
	position,
	type
FROM
	system_schema.columns
WHERE
	keyspace_name = ?
	AND table_name = ?;
`

type cassandraDatabaseColumn struct {
	databaseColumn

	// The position of the column for its type
	// If non-key: -1
	// If partition key: >= 0
	// If clustering key: >= 0
	Position int

	Kind string // The kind of the column (partition key, clustering key, regular)
}

type CassandraModule struct {
	pooler map[string]*gocql.Session // A pooler for each connection string
	mtx    *sync.RWMutex             // To protect the pooler from concurrent access
}

// Retrieve or create a connection from the pooler
func (m *CassandraModule) GetDBConnection(connectionString string) (*gocql.Session, error) {
	m.mtx.RLock()
	if session, ok := m.pooler[connectionString]; ok {
		m.mtx.RUnlock()
		return session, nil
	}
	m.mtx.RUnlock()

	// Parse the DSN
	if connectionString == "" {
		return nil, fmt.Errorf("connection string is empty")
	}

	// Create a new session and store it in the pooler
	// so that another table can use it
	//
	// Because Cassandra does not have a DSN, we parse the URL ourself
	parsed, err := url.Parse(connectionString)
	if err != nil {
		return nil, fmt.Errorf("error parsing the connection string: %v", err)
	}

	queryParams := parsed.Query()

	hosts := strings.Split(parsed.Host, ",")
	if len(hosts) == 0 || (hosts[0] == "" && len(hosts) == 1) {
		return nil, fmt.Errorf("no hosts found in the connection string")
	}

	// Trim spaces
	for i, host := range hosts {
		hosts[i] = strings.TrimSpace(host)
	}

	// Create a new Cassandra cluster configuration
	cluster := gocql.NewCluster(hosts...)

	// Check if the consistency level is set in the connection string
	if queryParams.Get("consistency") != "" {
		consistency, err := gocql.ParseConsistencyWrapper(queryParams.Get("consistency"))
		if err != nil {
			return nil, fmt.Errorf("error parsing consistency level: %v", err)
		}
		cluster.Consistency = consistency
	}

	// TLS settings
	if queryParams.Get("tls") == "true" || queryParams.Get("ssl") == "true" {
		cluster.SslOpts = &gocql.SslOptions{
			EnableHostVerification: true,
		}

		// Check if a CA certificate is provided
		certPath := queryParams.Get("tls_cert")
		if certPath != "" {
			cluster.SslOpts.CertPath = certPath
		}

		caPath := queryParams.Get("tls_ca_cert")
		if caPath != "" {
			cluster.SslOpts.CaPath = caPath
		}

		keyPath := queryParams.Get("tls_key")
		if keyPath != "" {
			cluster.SslOpts.KeyPath = keyPath
		}
	}

	// Check if a username and password are provided
	if parsed.User != nil {
		username := parsed.User.Username()
		password, _ := parsed.User.Password()
		if username != "" || password != "" {
			cluster.Authenticator = gocql.PasswordAuthenticator{
				Username: username,
				Password: password,
			}
		}
	}

	// Create the session
	session, err := cluster.CreateSession()
	if err != nil {
		return nil, fmt.Errorf("error creating a new Cassandra session: %v", err)
	}

	m.mtx.Lock()
	m.pooler[connectionString] = session
	m.mtx.Unlock()

	return session, nil
}

type CassandraTable struct {
	connection                  *gocql.Session
	tableName                   string
	schema                      []cassandraDatabaseColumn
	module                      *CassandraModule
	connectionString            string
	partitionClusteringKeyCount int // The number of clustering and partition keys in the table
	clusteringKeyCount          int // The number of partition keys in the table
}

type CassandraCursor struct {
	connection    *gocql.Session
	tableName     string
	schema        []cassandraDatabaseColumn
	iter          *gocql.Iter
	exhausted     bool
	currentRow    []interface{}
	columnsMapper map[string]int // A map to quickly access the column index by its name
	rowsReturned  int64
	limit         int64
	query         cassandraSQLQueryToExecute
}

type cassandraSQLQueryToExecute struct {
	SQLQueryToExecute

	// Additional fields specific to Cassandra

	// The column index corresponding for each argument in the query
	//
	// For example, if the first constraint is on the third column, ColumnIndex[0] = 2
	ColumnIndex []int
}

func (m *CassandraModule) Create(c *sqlite3.SQLiteConn, args []string) (sqlite3.VTab, error) {
	return m.Connect(c, args)
}

func (m *CassandraModule) TransactionModule() {}

func (v *CassandraModule) DestroyModule() {}

func (m *CassandraModule) Connect(c *sqlite3.SQLiteConn, args []string) (sqlite3.VTab, error) {
	// Init the structure
	if m.pooler == nil {
		m.pooler = make(map[string]*gocql.Session)
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
	session, err := m.GetDBConnection(connectionString)
	if err != nil {
		return nil, fmt.Errorf("error opening a connection to the database: %v", err)
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
	query := session.Query(fetchCassandraSchemaSQLQuery, schemaName, table)

	iter := query.Iter()
	if iter == nil {
		return nil, fmt.Errorf("error fetching the schema for the table %s.%s", schemaName, table)
	}

	// Iterate over the rows and create the schema
	internalSchema := []cassandraDatabaseColumn{}
	schema := strings.Builder{}
	schema.WriteString("CREATE TABLE x(\n")

	scanner := iter.Scanner()

	partitionClusteringKeyCount := 0
	clusteringKeyCount := 0
	for scanner.Next() {
		var columnName, kind, dataType string
		var position int
		err = scanner.Scan(&columnName, &kind, &position, &dataType)
		if err != nil {
			return nil, fmt.Errorf("error scanning the schema: %v", err)
		}
		dataType = strings.TrimSpace(dataType)
		dataType = strings.ToLower(dataType)

		if strings.HasPrefix(dataType, "frozen<") {
			// Remove the frozen< and >
			dataType = strings.TrimPrefix(dataType, "frozen<")
			dataType = strings.TrimSuffix(dataType, ">")
		}

		// If the column has a tuple, we don't support it
		// We skip them as long as they are weirdly handled by gocql
		//
		// Currently, rather than returning a []interface{} for the tuple, it splits the tuple into multiple columns
		// named tuple_column_name[index]
		if strings.Contains(dataType, "tuple<") {
			// In case the column is a partition key or clustering key, we return an error as we'll never be able to query it
			if kind == "partition_key" || kind == "clustering" {
				return nil, fmt.Errorf("anyquery does not support tuples as partition or clustering keys. Please remove the tuple from the schema")
			}
			continue
		}

		columnType := "TEXT"
		typeSupported := false
		var defaultValue interface{}
		switch {
		case dataType == "int" || dataType == "bigint" || dataType == "smallint" ||
			dataType == "tinyint" || strings.Contains(dataType, "varint") || dataType == "counter":
			columnType = "INTEGER"
			typeSupported = true
			defaultValue = 0
		case dataType == "float" || dataType == "double" || dataType == "decimal" ||
			dataType == "single" || strings.HasPrefix(dataType, "decimal"):
			columnType = "REAL"
			typeSupported = true
			defaultValue = 0.0

		case dataType == "ascii" || dataType == "inet" || dataType == "text" || dataType == "varchar" ||
			dataType == "uuid" || dataType == "timeuuid" || dataType == "duration":
			columnType = "TEXT"
			typeSupported = true
			defaultValue = ""

		case dataType == "timestamp":
			columnType = "DATETIME"
			typeSupported = true
			defaultValue = ""

		case dataType == "date":
			columnType = "DATE"
			typeSupported = true
			defaultValue = ""
		case dataType == "blob":
			columnType = "BLOB"
			typeSupported = true
			defaultValue = []byte{}

		case strings.Contains(dataType, "list<") || strings.Contains(dataType, "set<") ||
			strings.Contains(dataType, "map<") || strings.Contains(dataType, "tuple<") ||
			strings.Contains(dataType, "udt<") || strings.Contains(dataType, "frozen<"):
			columnType = "JSON"
			typeSupported = true
			defaultValue = "{}"

		case dataType == "boolean":
			columnType = "BOOLEAN"
			typeSupported = true
			defaultValue = false

		default:
			columnType = "UNKNOWN"
			typeSupported = false // Fail safe
			defaultValue = nil
		}

		switch kind {
		case "partition_key":
			partitionClusteringKeyCount++
		case "clustering":
			partitionClusteringKeyCount++
			clusteringKeyCount++
		}

		localColumnName := transformSQLiteValidName(columnName)

		internalSchema = append(internalSchema, cassandraDatabaseColumn{
			databaseColumn: databaseColumn{
				SQLiteName:   localColumnName,
				Realname:     columnName,
				Type:         columnType,
				RemoteType:   dataType,
				Supported:    typeSupported,
				DefaultValue: defaultValue,
			},
			Position: position,
			Kind:     kind,
		})

	}

	if scanner.Err() != nil {
		return nil, fmt.Errorf("error iterating over the rows: %v", scanner.Err())
	}

	// Sort the schema where the partition key is first, then clustering keys by order, and leave the rest
	slices.SortStableFunc(internalSchema, func(a, b cassandraDatabaseColumn) int {
		switch {
		case a.Kind == b.Kind:
			// If both are the same kind, sort by position
			if a.Position == b.Position {
				return 0
			} else if a.Position < b.Position {
				return -1
			} else {
				return 1
			}
		case a.Kind == "partition_key" && b.Kind != "partition_key":
			// If a is a partition key and b is not, a comes first
			return -1
		case a.Kind != "partition_key" && b.Kind == "partition_key":
			// If b is a partition key and a is not, b comes first
			return 1
		case a.Kind == "clustering" && b.Kind != "clustering":
			// If a is a clustering key and b is not, a comes first
			return -1
		case a.Kind != "clustering" && b.Kind == "clustering":
			// If b is a clustering key and a is not, b comes first
			return 1
		default: // Should not happen
			return 0
		}
	})

	// Write the schema to the CREATE TABLE statement
	for i, col := range internalSchema {
		if i > 0 {
			schema.WriteString(",\n")
		}
		schema.WriteString(fmt.Sprintf("  `%s` %s", col.SQLiteName, col.Type))
	}

	schema.WriteString("\n)")

	if len(internalSchema) == 0 {
		return nil, fmt.Errorf("no columns found for the table. Make sure the table exists and has columns")
	}

	// Declare the virtual table
	err = c.DeclareVTab(schema.String())
	if err != nil {
		return nil, fmt.Errorf("error declaring the virtual table: %v", err)
	}

	// Merge the schema with the table name
	table = fmt.Sprintf("\"%s\".\"%s\"", schemaName, table)

	// Return the table instance
	return &CassandraTable{
		tableName:                   table,
		schema:                      internalSchema,
		module:                      m,
		connectionString:            connectionString,
		partitionClusteringKeyCount: partitionClusteringKeyCount,
		clusteringKeyCount:          clusteringKeyCount,
	}, nil
}

func (t *CassandraTable) Open() (sqlite3.VTabCursor, error) {
	// Create a new connection for each cursor
	session, err := t.module.GetDBConnection(t.connectionString)
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
				values[i] = new(NullUint64) // Cassandra does not have unsigned integers, so we use NullInt64
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

	return &CassandraCursor{
		connection: session,
		tableName:  t.tableName,
		schema:     t.schema,
		limit:      -1,
		currentRow: values,
	}, nil
}

func (t *CassandraTable) Disconnect() error {
	return t.Destroy()
}

func (t *CassandraTable) releaseConnection() {
	if t.connection != nil {
		t.connection.Close()
		t.connection = nil
	}
}

func (t *CassandraTable) Destroy() error {
	// Release the connection
	t.releaseConnection()
	return nil
}

// Returns true if all the values in the slice are true
func slicesAllTrue(s []bool) bool {
	for _, v := range s {
		if !v {
			return false
		}
	}
	return true
}

// A function that will be called several times to check the best way to access the data.
// This function is called with different constraints and order by clauses
//
// To find the method, we will ask the database to explain the query and return the best method
func (t *CassandraTable) BestIndex(cst []sqlite3.InfoConstraint, ob []sqlite3.InfoOrderBy, info sqlite3.IndexInformation) (*sqlite3.IndexResult, error) {
	// This function is called to find the best way to query the database
	// Because Cassandra has tons of limitations, we'll try to recommend SQLite to uses query that filters
	// on the partition key and clustering key, and to use the LIMIT and OFFSET clauses
	//
	// A query starts with a cost of 1 000 000. Each possible filter will reduce the cost by 0.9x
	// To figure out the possible filters, we construct a []bool that indicates which constraints are used

	used := make([]bool, len(cst))
	alreadyOrdered := false
	cost := 1000000.0

	// A slice to hold the partition and clustering keys that have a WHERE clause
	partitionClusteringKeys := make([]bool, t.partitionClusteringKeyCount)
	limitCstIndex := -1
	offsetCstIndex := -1

	for _, c := range cst {
		if c.Column >= len(t.schema) {
			continue // Ignore constraints on non-existing columns
		}
		col := t.schema[c.Column]

		switch col.Kind {
		case "partition_key":
			if c.Op == sqlite3.OpEQ {
				partitionClusteringKeys[c.Column] = true // We can use this partition key in the query
			}
		case "clustering":
			if c.Op == sqlite3.OpEQ || c.Op == sqlite3.OpLT || c.Op == sqlite3.OpGT ||
				c.Op == sqlite3.OpLE || c.Op == sqlite3.OpGE {
				partitionClusteringKeys[c.Column] = true
			}
		}
	}

	// Now we are going to mark the constraints that can be used following the constraints of CQL
	for i, c := range cst {
		if c.Column >= len(t.schema) {
			continue // Ignore constraints on non-existing columns
		}
		col := t.schema[c.Column]

		// We can use a partition key ONLY if it has an equality constraint and all previous partition keys have equality constraints
		if col.Kind == "partition_key" && c.Op == sqlite3.OpEQ && slicesAllTrue(partitionClusteringKeys[:c.Column]) {
			used[i] = true
			cost *= 0.9 // Reduce the cost by 10%
			continue
		}

		if col.Kind == "clustering" && slicesAllTrue(partitionClusteringKeys[:c.Column]) {
			// We can use a clustering key if all previous partition keys and clustering keys have equality constraints
			if c.Op == sqlite3.OpEQ || c.Op == sqlite3.OpLT || c.Op == sqlite3.OpGT ||
				c.Op == sqlite3.OpLE || c.Op == sqlite3.OpGE {
				used[i] = true
				cost *= 0.9 // Reduce the cost by 10%
				continue
			}
		}

		// If the constraint is a LIMIT, we can use it
		if c.Op == sqlite3.OpLIMIT {
			limitCstIndex = i
			used[i] = true
			cost *= 0.9 // Reduce the cost by 10%
			continue
		}

		// An offset cannot be used in Cassandra, so we ignore it
	}

	// Now, we do literrally the same thing for the ORDER BY clauses
	clusteringKeysUsedOrder := make([]bool, t.clusteringKeyCount)
	partitionKeyCount := t.partitionClusteringKeyCount - t.clusteringKeyCount
	for _, o := range ob {
		if o.Column >= len(t.schema) {
			continue // Ignore order by on non-existing columns
		}
		col := t.schema[o.Column]
		if col.Kind == "clustering" {
			clusteringKeysUsedOrder[o.Column-partitionKeyCount] = true
			cost *= 0.9
		}
	}

	// For all ORDER BY clauses, we check if they are supported
	alreadyOrdered = true
	for _, o := range ob {
		if o.Column >= len(t.schema) {
			continue // Ignore order by on non-existing columns
		}

		col := t.schema[o.Column]
		if col.Kind != "clustering" {
			alreadyOrdered = false
			break
		}

		if !slicesAllTrue(clusteringKeysUsedOrder[:o.Column-partitionKeyCount]) {
			alreadyOrdered = false
			break
		}
	}

	// Now, let's build the query
	builder := sqlbuilder.NewSelectBuilder()
	cols := []string{}
	for i, col := range t.schema {
		if info.ColUsed&(1<<i) == 0 && i < 62 {
			continue // Skip columns that are not used in the query
		}
		cols = append(cols, col.Realname)
	}

	// If no columns are used, we select the first column
	if len(cols) == 0 {
		cols = append(cols, t.schema[0].Realname)
	}

	builder.Select(cols...).From(t.tableName)

	columnIndexes := make([]int, 0, len(cst))

	andConditions := []string{}
	for i, c := range cst {
		if !used[i] {
			continue
		}

		switch c.Op {
		case sqlite3.OpEQ:
			andConditions = append(andConditions, builder.Equal(t.schema[c.Column].Realname, t.schema[c.Column].DefaultValue))
			columnIndexes = append(columnIndexes, c.Column)
		case sqlite3.OpGT:
			andConditions = append(andConditions, builder.GreaterThan(t.schema[c.Column].Realname, t.schema[c.Column].DefaultValue))
			columnIndexes = append(columnIndexes, c.Column)
		case sqlite3.OpGE:
			andConditions = append(andConditions, builder.GreaterEqualThan(t.schema[c.Column].Realname, t.schema[c.Column].DefaultValue))
			columnIndexes = append(columnIndexes, c.Column)
		case sqlite3.OpLT:
			andConditions = append(andConditions, builder.LessThan(t.schema[c.Column].Realname, t.schema[c.Column].DefaultValue))
			columnIndexes = append(columnIndexes, c.Column)
		case sqlite3.OpLE:
			andConditions = append(andConditions, builder.LessEqualThan(t.schema[c.Column].Realname, t.schema[c.Column].DefaultValue))
			columnIndexes = append(columnIndexes, c.Column)

		}
	}
	if len(andConditions) > 0 {
		builder.Where(andConditions...)
	}

	builder.SetFlavor(sqlbuilder.CQL)

	orders := []string{}
	if alreadyOrdered {
		for _, o := range ob {
			if o.Desc {
				orders = append(orders, fmt.Sprintf("%s DESC", t.schema[o.Column].Realname))
			} else {
				orders = append(orders, fmt.Sprintf("%s ASC", t.schema[o.Column].Realname))
			}
		}

		if len(orders) > 0 {
			builder.OrderBy(orders...)
		}
	}

	// Build the query and its arguments
	rawQuery, args := builder.Build()
	query := &cassandraSQLQueryToExecute{
		SQLQueryToExecute: SQLQueryToExecute{
			Query:       rawQuery,
			Args:        args,
			LimitIndex:  limitCstIndex,
			OffsetIndex: offsetCstIndex,
			ColumnsUsed: info.ColUsed,
		},
		ColumnIndex: columnIndexes,
	}

	// Serialize the query as a JSON object
	serializedQuery, err := json.Marshal(query)
	if err != nil {
		return nil, fmt.Errorf("error serializing the query: %v", err)
	}

	return &sqlite3.IndexResult{
		Used:           used,
		IdxStr:         string(serializedQuery),
		AlreadyOrdered: alreadyOrdered,
		EstimatedCost:  cost,
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

func (t *CassandraTable) Insert(id any, vals []any) (int64, error) {
	return 0, fmt.Errorf("insert operation is not supported for Cassandra tables")
}

func (t *CassandraTable) Update(id any, vals []any) error {
	return fmt.Errorf("update operation is not supported for Cassandra tables")
}

func (t *CassandraTable) Delete(id any) error {
	return fmt.Errorf("delete operation is not supported for Cassandra tables")
}

func (t *CassandraTable) PartialUpdate() bool {
	return false
}

func (t *CassandraCursor) resetCursor() error {
	if t.iter != nil {
		t.iter.Close()
		t.iter = nil
	}
	t.limit = -1
	t.rowsReturned = 0
	t.exhausted = false

	return nil
}

func (t *CassandraCursor) Filter(idxNum int, idxStr string, vals []interface{}) error {
	// Reset the cursor as Filter might be called multiple times
	err := t.resetCursor()
	if err != nil {
		return fmt.Errorf("error resetting the cursor: %v", err)
	}

	// Reconstruct the query and its arguments
	var query cassandraSQLQueryToExecute
	err = json.Unmarshal([]byte(idxStr), &query)
	if err != nil {
		return fmt.Errorf("error unmarshalling the query: %v", err)
	}

	// Set the query for the cursor
	t.query = query

	queryParams := []interface{}{}
	limit := int64(-1) // Default limit is -1 (no limit)
	for i, c := range vals {
		if i == query.LimitIndex {
			limit = c.(int64)
		} else {
			// In case the column is JSON (list, map, set, etc.), the user pass the value as a string (because SQLite does not support JSON natively)
			// but go-cql expects a slice, or a map
			//
			// Therefore, when we encounter a JSON column, we need to parse the value using the json.Unmarshal function
			if t.schema[query.ColumnIndex[i]].Type == "JSON" {
				// Ensure the value is a string
				if strVal, ok := c.(string); ok {
					var jsonValue interface{}
					err := json.Unmarshal([]byte(strVal), &jsonValue)
					if err != nil {
						return fmt.Errorf("error unmarshalling the JSON value for column %s with value %s: %v", t.schema[query.ColumnIndex[i]].Realname, strVal, err)
					}
					queryParams = append(queryParams, jsonValue)
				} else {
					return fmt.Errorf("expected a string for JSON column %s, got %T", t.schema[query.ColumnIndex[i]].Realname, c)
				}
			} else {
				queryParams = append(queryParams, c)
			}
		}
	}

	// Add the LIMIT to the query
	if limit != -1 {
		query.Query += fmt.Sprintf(" LIMIT %d", limit)
		t.limit = limit
	}

	cassandraQuery := t.connection.Query(query.Query, queryParams...)

	t.iter = cassandraQuery.Iter()
	if t.iter == nil {
		return fmt.Errorf("error creating the iterator for the query")
	}
	return t.Next()
}

func (t *CassandraCursor) Next() error {
	defer func() {
		t.rowsReturned++
	}()
	if t.iter == nil {
		return fmt.Errorf("no rows to iterate over")
	}

	line, err := t.iter.RowData()
	if err != nil {
		return fmt.Errorf("error getting the row data: %v", err)
	}

	if !t.iter.Scan(line.Values...) {
		// Cursor is exhausted
		t.currentRow = nil
		t.exhausted = true
		return nil
	}

	t.currentRow = line.Values

	if t.columnsMapper == nil {
		// Initialize the columns mapper
		t.columnsMapper = make(map[string]int, len(t.schema))
		for i, col := range line.Columns {
			t.columnsMapper[col] = i
		}
	}
	if t.limit != -1 && t.rowsReturned >= t.limit {
		// If a limit is set, we stop here
		t.exhausted = true
	}

	return nil
}

func (t *CassandraCursor) Column(context *sqlite3.SQLiteContext, col int) error {
	if col < 0 || col >= len(t.schema) {
		context.ResultNull()
		return nil
	}

	if t.currentRow == nil {
		context.ResultNull()
	}

	colInfo := t.schema[col]
	colPos, ok := t.columnsMapper[colInfo.Realname]
	if colPos < 0 || colPos >= len(t.currentRow) || !ok {
		context.ResultNull()
		return nil
	}

	unparsed, ok := t.currentRow[colPos].(*interface{})
	if !ok || unparsed == nil {
		context.ResultNull()
		return nil
	}

	parsed := *unparsed

	switch val := parsed.(type) {
	case nil:
		context.ResultNull()
	case string:
		context.ResultText(val)
	case []byte:
		context.ResultBlob(val)
	case int:
		context.ResultInt64(int64(val))
	case int8:
		context.ResultInt64(int64(val))
	case int16:
		context.ResultInt64(int64(val))
	case int32:
		context.ResultInt64(int64(val))
	case int64:
		context.ResultInt64(val)
	case float32:
		context.ResultDouble(float64(val))
	case float64:
		context.ResultDouble(val)

	case bool:
		context.ResultBool(val)
	case net.IP:
		context.ResultText(val.String())
	case *big.Int:
		// Convert the big.Int to a string
		context.ResultInt64(val.Int64())
	case gocql.UUID:
		context.ResultText(val.String())
	case *inf.Dec:
		// Convert the inf.Dec to a string
		context.ResultText(val.String())
	case gocql.Duration:
		// Convert the duration to int64 nanoseconds
		// We consider a month as 30 days for simplicity
		duration := int64(val.Months)*30*24*time.Hour.Nanoseconds() +
			int64(val.Days)*24*time.Hour.Nanoseconds() +
			val.Nanoseconds
		context.ResultInt64(duration)

	case time.Duration:
		// Convert the duration to int64 nanoseconds
		context.ResultInt64(val.Nanoseconds())
	case time.Time:
		if val.IsZero() {
			context.ResultNull()
			return nil
		}
		context.ResultText(val.Format(time.RFC3339))

	case []interface{}:
		context.ResultNull()

	default:
		// []interface{} is also used as a NULL value for gocql
		if arr, ok := val.([]interface{}); ok {
			if len(arr) == 0 {
				context.ResultNull()
				return nil
			}
			// Otherwise, we assume it's a JSON object
		}

		// Convert to JSON
		jsonVal, err := json.Marshal(val)
		if err != nil {
			context.ResultNull()
			return nil
		}
		context.ResultText(string(jsonVal))
	}

	return nil
}

func (t *CassandraCursor) EOF() bool {
	return t.exhausted
}

func (t *CassandraCursor) Rowid() (int64, error) {
	return rand.Int64(), nil
}

func (t *CassandraCursor) Close() error {
	return nil
}
