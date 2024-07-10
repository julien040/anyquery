package module

import (
	"encoding/json"
	"errors"
	"io"
	rand "math/rand/v2"
	"reflect"
	"strings"
	"time"

	"github.com/gammazero/deque"
	"github.com/hashicorp/go-hclog"
	"github.com/julien040/anyquery/rpc"
	"github.com/mattn/go-sqlite3"
)

const (
	minimumCapacityRingBuffer = 256
	preAllocatedCapacity      = 128
)

// This file links the plugin to the SQLite Virtual Table interface

// SQLiteModule is a struct that holds the information about the SQLite module
//
// For each table that the plugin provides and for each profile, a new SQLiteModule
// should be created and registered in the main program
type SQLiteModule struct {
	PluginPath      string
	PluginArgs      []string
	PluginManifest  rpc.PluginManifest
	ConnectionIndex int
	TableIndex      int
	client          *rpc.InternalClient
	UserConfig      rpc.PluginConfig
	Logger          hclog.Logger
	ConnectionPool  *rpc.ConnectionPool
	Stderr          io.Writer
}

// SQLiteTable that holds the information needed for the BestIndex and Open methods
type SQLiteTable struct {
	PluginPath              string
	connectionIndex         int
	nextCursor              int
	tableIndex              int
	schema                  rpc.DatabaseSchema
	client                  *rpc.InternalClient
	ConnectionPool          *rpc.ConnectionPool
	insertBuffer            *deque.Deque[[]interface{}]
	maxBufferInsert         uint
	updateBuffer            *deque.Deque[updateItem]
	maxBufferUpdate         uint
	deleteBuffer            *deque.Deque[interface{}]
	maxBufferDelete         uint
	mapColPositionColPlugin map[int]int // Map the position of the column in SQLite to the position of the column in the rows returned by the plugin
}

// SQLiteCursor holds the information needed for the Column, Filter, EOF and Next methods
type SQLiteCursor struct {
	connectionIndex         int
	tableIndex              int
	cursorIndex             int
	schema                  rpc.DatabaseSchema
	client                  *rpc.InternalClient
	noMoreRows              bool
	rows                    *deque.Deque[[]interface{}] // A ring buffer to store the rows before sending them to SQLite
	nextCursor              *int
	constraints             rpc.QueryConstraint
	mapColPositionColPlugin map[int]int // Map the position of the column in SQLite to the position of the column in the rows returned by the plugin
}

type updateItem struct {
	id   any
	vals []any
}

// EponymousOnlyModule is a method that is used to mark the table as eponymous-only
//
// See https://www.sqlite.org/vtab.html#eponymous_virtual_tables for more information
func (m *SQLiteModule) EponymousOnlyModule() {}

// Create is called when the virtual table is created e.g. CREATE VIRTUAL TABLE or SELECT...FROM(epo_table)
//
// Its main job is to create a new RPC client and return the needed information
// for the SQLite virtual table methods
func (m *SQLiteModule) Create(c *sqlite3.SQLiteConn, args []string) (sqlite3.VTab, error) {
	// Create a new plugin instance
	// and store the client in the module
	rpcClient, err := m.ConnectionPool.NewClient(rpc.NewClientParams{
		ExecutableLocation: m.PluginPath,
		ExecutableArg:      m.PluginArgs,
		Logger:             m.Logger,
		Stderr:             m.Stderr,
	})
	if err != nil {
		return nil, errors.Join(errors.New("could not create a new rpc client for "+m.PluginPath), err)
	}
	m.client = rpcClient

	// Request the schema of the table from the plugin
	dbSchema, err := m.client.Plugin.Initialize(m.ConnectionIndex, m.TableIndex, m.UserConfig)
	if err != nil {
		return nil, errors.Join(errors.New("could not request the schema of the table from the plugin "+m.PluginPath), err)
	}

	// Verify that the schema is correct
	if len(dbSchema.Columns) == 0 {
		return nil, errors.New("the schema of the table is empty")
	}

	// Create the schema in SQLite
	stringSchema := createSQLiteSchema(dbSchema)
	err = c.DeclareVTab(stringSchema)
	if err != nil {
		return nil, errors.Join(errors.New("could not declare the virtual table in SQLite"), err, errors.New("Schema: "+stringSchema))
	}

	// Compute the mapColPositionColPlugin so that we can map the position of the column in SQLite
	// to the position of the column in the rows returned by the plugin
	//
	// We don't have a 1:1 map because some columns are parameters and are not returned by the plugin
	colMapper := make(map[int]int)
	drift := 0
	for i, col := range dbSchema.Columns {
		if col.IsParameter {
			drift++
		} else {
			colMapper[i] = i - drift
		}
	}

	// Initialize a new table
	table := &SQLiteTable{
		m.PluginPath,
		m.ConnectionIndex,
		0,
		m.TableIndex,
		dbSchema,
		m.client,
		m.ConnectionPool,
		&deque.Deque[[]interface{}]{},
		dbSchema.BufferInsert,
		&deque.Deque[updateItem]{},
		dbSchema.BufferUpdate,
		&deque.Deque[interface{}]{},
		dbSchema.BufferDelete,
		colMapper,
	}

	return table, nil
}

// createSQLiteSchema creates the schema of the table in SQLite
// using the sqlite3.SQLiteConn.DeclareVTab method
func createSQLiteSchema(arg rpc.DatabaseSchema) string {
	// Initialize a string builder to efficiently create the schema
	var schema strings.Builder

	// The table name is not important, we set it to x therefore
	schema.WriteString("CREATE TABLE x(")

	// We iterate over the columns and add them to the schema
	for i, col := range arg.Columns {
		// To escape the column name, we wrap it in double quotes
		// and replace any double quote in the column name with two double quotes
		schema.WriteRune('"')
		schema.WriteString(strings.ReplaceAll(col.Name, `"`, `""`))
		schema.WriteRune('"')
		schema.WriteByte(' ')
		switch col.Type {
		case rpc.ColumnTypeInt:
			schema.WriteString("INTEGER")
		case rpc.ColumnTypeString:
			schema.WriteString("TEXT")
		case rpc.ColumnTypeBlob:
			schema.WriteString("BLOB")
		case rpc.ColumnTypeFloat:
			schema.WriteString("REAL")
		}

		// If the column is a parameter, we add the HIDDEN keyword
		if col.IsParameter {
			schema.WriteString(" HIDDEN")
		}

		// If the column is the primary key, we add the PRIMARY KEY keyword
		if i == arg.PrimaryKey {
			schema.WriteString(" PRIMARY KEY")
		}

		// We add a comma if it's not the last column
		if i != len(arg.Columns)-1 {
			schema.WriteString(", ")
		}
	}
	// We close the schema
	schema.WriteRune(')')

	// We check if the plugin has a primary key
	// If so, we add "WITHOUT ROWID" to the schema
	if arg.PrimaryKey != -1 {
		schema.WriteString(" WITHOUT ROWID")
	}

	// Add the last semicolon
	schema.WriteRune(';')

	// We declare the virtual table in SQLite
	return schema.String()
}

// Connect is called when the virtual table is connected
//
// Because it's an eponymous-only module, the method must be identical to Create
func (m *SQLiteModule) Connect(c *sqlite3.SQLiteConn, args []string) (sqlite3.VTab, error) {
	return m.Create(c, args)
}

// BestIndex is called when the virtual table is queried
// to figure out the best way to query the table
//
// However, we don't use it that way but only to serialize the constraints
// for the Filter method
func (t *SQLiteTable) BestIndex(cst []sqlite3.InfoConstraint, ob []sqlite3.InfoOrderBy) (*sqlite3.IndexResult, error) {
	// The first task of BestIndex is to check if the required parameters are present
	// If not, we return sqlite3.ErrConstraint
	present := make([]bool, len(t.schema.Columns))
	for _, c := range cst {
		if c.Usable && c.Op == sqlite3.OpEQ {
			present[c.Column] = true
		}
	}
	for i, col := range t.schema.Columns {
		if col.IsRequired && !present[i] {
			return nil, sqlite3.ErrConstraint
		}
	}

	// We serialize the constraints so that we can pass them to the Filter method
	// The only way to communicate them to the Filter method is through the IdxStr field
	// Therefore, we must serialize them as JSON and unmarshal them in the Filter method
	constraints := rpc.QueryConstraint{
		Limit:  -1,
		Offset: -1,
	}

	// Used is a boolean array that tells SQLite which constraints are used
	// and that must be passed to the Filter method in the vals field
	used := make([]bool, len(cst))
	parseConstraintsFromSQLite(cst, ob, &constraints, used, t.schema)

	// We store the constraints as JSON to be passed with IdxStr in IndexResult
	marshal, err := json.Marshal(constraints)
	if err != nil {
		return nil, errors.Join(errors.New("could not marshal the constraints"), err)
	}

	return &sqlite3.IndexResult{
		IdxNum: 0,
		IdxStr: string(marshal),
		Used:   used,
	}, nil

}

// Open is called when a new cursor is opened
//
// It should return a new cursor
func (t *SQLiteTable) Open() (sqlite3.VTabCursor, error) {
	// For coherence, we flush the buffers of inserts, updates and deletes
	// before running any SELECT query
	// If any of the flush fails, we return an error and therefore stop the query

	err := t.flushInsert()
	if err != nil {
		return nil, errors.Join(errors.New("recent inserts were not saved, and an attempt to flush them failed. Please retry the query"), err)
	}
	err = t.flushUpdate()
	if err != nil {
		return nil, errors.Join(errors.New("recent updates were not saved, and an attempt to flush them failed. Please retry the query"), err)
	}
	err = t.flushDelete()
	if err != nil {
		return nil, errors.Join(errors.New("recent deletes were not saved, and an attempt to flush them failed. Please retry the query"), err)
	}

	// We create a new cursor
	cursor := &SQLiteCursor{
		t.connectionIndex,
		t.tableIndex,
		t.nextCursor,
		t.schema,
		t.client,
		false,
		deque.New[[]interface{}](preAllocatedCapacity, minimumCapacityRingBuffer),
		&t.nextCursor,
		rpc.QueryConstraint{},
		t.mapColPositionColPlugin,
	}
	// We increment the cursor id for the next cursor by 1
	// so that the next cursor will have a different id
	t.nextCursor++

	return cursor, nil
}

func (t *SQLiteTable) Insert(id any, vals []any) (int64, error) {
	if t.schema.PrimaryKey == -1 {
		return 0, errors.New("the table does not support INSERT because it has no primary key")
	}

	if !t.schema.HandlesInsert {
		return 0, errors.New("the table does not support INSERT")
	}

	// We add the row to the buffer
	t.insertBuffer.PushBack(vals)

	// If the buffer is full, we flush it
	if uint(t.insertBuffer.Len()) >= t.maxBufferInsert {
		err := t.flushInsert()
		if err != nil {
			return 0, errors.Join(errors.New("could not flush the insert buffer"), err)
		}
	}

	// We return a random number as the row ID
	// but it's a fail safe because a table that supports INSERT must have a primary key
	// and therefore no row ID is needed
	randomID := rand.Int64()

	return randomID, nil
}

func (t *SQLiteTable) Update(id any, vals []any) error {
	if t.schema.PrimaryKey == -1 {
		return errors.New("the table does not support UPDATE because it has no primary key")
	}

	if !t.schema.HandlesUpdate {
		return errors.New("the table does not support UPDATE")
	}

	t.updateBuffer.PushBack(updateItem{id, vals})

	if uint(t.updateBuffer.Len()) >= t.maxBufferUpdate {
		err := t.flushUpdate()
		if err != nil {
			return errors.Join(errors.New("could not flush the update buffer"), err)
		}
	}

	return nil
}

func (t *SQLiteTable) Delete(id any) error {
	if t.schema.PrimaryKey == -1 {
		return errors.New("the table does not support DELETE because it has no primary key")
	}

	if !t.schema.HandlesDelete {
		return errors.New("the table does not support DELETE")
	}

	t.deleteBuffer.PushBack(id)

	if uint(t.deleteBuffer.Len()) >= t.maxBufferDelete {
		err := t.flushDelete()
		if err != nil {
			return errors.Join(errors.New("could not flush the delete buffer"), err)
		}
	}
	return nil
}

// Flush the buffer of inserts to the plugin
//
// If the plugin rejects the inserts, the functions returns an error
// and keeps the rows in the buffer for a later retry
func (t *SQLiteTable) flushInsert() error {
	if t.insertBuffer.Len() == 0 {
		return nil
	}

	// We request the plugin to insert the rows
	rows := make([][]interface{}, t.insertBuffer.Len())
	for i := 0; i < t.insertBuffer.Len(); i++ {
		rows[i] = t.insertBuffer.At(i)
	}
	err := t.client.Plugin.Insert(t.connectionIndex, t.tableIndex, rows)
	if err != nil {
		return errors.Join(errors.New("could not insert the rows in the plugin"), err)
	}

	// We clear the buffer
	t.insertBuffer.Clear()
	return nil
}

// Flush the buffer of updates to the plugin
//
// If the plugin rejects the updates, the functions returns an error
// and keeps the rows in the buffer for a later retry
func (t *SQLiteTable) flushUpdate() error {
	if t.updateBuffer.Len() == 0 {
		return nil
	}

	// We request the plugin to update the rows
	rows := make([][]interface{}, t.updateBuffer.Len())
	// The format for a row update is [id, ...vals]
	for i := 0; i < t.updateBuffer.Len(); i++ {
		item := t.updateBuffer.At(i)
		rows[i] = append([]interface{}{item.id}, item.vals...)
	}

	err := t.client.Plugin.Update(t.connectionIndex, t.tableIndex, rows)
	if err != nil {
		return errors.Join(errors.New("could not update the rows in the plugin"), err)
	}

	// We clear the buffer
	t.updateBuffer.Clear()

	return nil
}

// Flush the buffer of deletes to the plugin
//
// If the plugin rejects the deletes, the functions returns an error
// and keeps the rows in the buffer for a later retry
func (t *SQLiteTable) flushDelete() error {
	if t.deleteBuffer.Len() == 0 {
		return nil
	}

	// We request the plugin to delete the rows
	rows := make([]interface{}, t.deleteBuffer.Len())
	for i := 0; i < t.deleteBuffer.Len(); i++ {
		rows[i] = t.deleteBuffer.At(i)
	}

	err := t.client.Plugin.Delete(t.connectionIndex, t.tableIndex, rows)
	if err != nil {
		return errors.Join(errors.New("could not delete the rows in the plugin"), err)
	}

	// We clear the buffer
	t.deleteBuffer.Clear()

	return nil
}

// Close is called when the cursor is no longer needed
func (c *SQLiteCursor) Close() error { return nil }

// These methods are not used in this plugin
func (v *SQLiteTable) Disconnect() error {
	// Flush the buffers before closing the client
	v.flushInsert()
	v.flushUpdate()
	v.flushDelete()
	time.Sleep(30 * time.Millisecond)
	// We close the client
	v.ConnectionPool.CloseConnection(v.PluginPath, v.connectionIndex)
	return nil
}
func (v *SQLiteTable) Destroy() error {
	return v.Disconnect()
}
func (v *SQLiteModule) DestroyModule() {
	// When a plugin was wrongly initialized and the module is destroyed
	// destroyModule is called rather than Disconnect
	v.ConnectionPool.CloseConnection(v.PluginPath, v.ConnectionIndex)
}

// Column is called when a column is queried
//
// It should return the value of the column
func (c *SQLiteCursor) Column(context *sqlite3.SQLiteContext, col int) error {
	// First, we need to check if the column is a parameter
	// If so, we return the value of the linked constraint
	// becase it must be the same value for all the rows
	if c.schema.Columns[col].IsParameter {
		// We find the constraint that is linked to the column
		// and return its value
		for _, cst := range c.constraints.Columns {
			if cst.ColumnID == col {
				convertToSQLiteVal(cst.Value, context)
				return nil
			}
		}
	} else {
		// If column is called with an empty row, we return NULL (fail safe)
		if c.rows.Len() == 0 {
			context.ResultNull()
		}
		// Get the position of the column in the rows returned by the plugin
		// and return the value of the column
		pluginCol, ok := c.mapColPositionColPlugin[col]
		if !ok {
			// If the column is not found in the map, we return NULL
			context.ResultNull()
		}

		// Otherwise, we return the value of the column from the ring buffer
		if len(c.rows.Front()) <= pluginCol {
			// The plugin did not return enough columns. We return NULL
			// TODO: Must return a log message
			context.ResultNull()
		} else {
			convertToSQLiteVal(c.rows.Front()[pluginCol], context)
		}
	}

	return nil
}

// convertToSQLiteVal asserts the type of the value and converts it to the SQLite type
func convertToSQLiteVal(val interface{}, c *sqlite3.SQLiteContext) {
	// We convert the value to the SQLite type
	// and store it in the SQLite context
	switch v := val.(type) {
	case string:
		c.ResultText(v)
	case int:
		c.ResultInt(v)
	case int8:
		c.ResultInt(int(v))
	case int16:
		c.ResultInt(int(v))
	case int32:
		c.ResultInt(int(v))
	case uint:
		c.ResultInt(int(reflect.ValueOf(v).Uint()))
	case uint8:
		c.ResultInt(int(v))
	case uint16:
		c.ResultInt(int(v))
	case uint32:
		c.ResultInt(int(v))
	case []uint, []uint8:
		c.ResultBlob(reflect.ValueOf(v).Bytes())
	case int64:
		c.ResultInt64(v)
	case uint64:
		c.ResultInt64(int64(v))
	case bool:
		if v {
			c.ResultInt(1)
		} else {
			c.ResultInt(0)
		}
	case float64:
		c.ResultDouble(v)
	case float32:
		c.ResultDouble(float64(v))
	case []string, []float64, []float32, []int, []int64, []bool, []interface{}, []uint64,
		[]uint32, []uint16, []int32, []int16, []int8:
		// JSON encode the string slice
		encoded, err := json.Marshal(v)
		if err != nil {
			// If the JSON encoding fails, we return NULL
			c.ResultNull()
		} else {
			c.ResultText(string(encoded))
		}

	case nil:
		c.ResultNull()
	default:
		// Try to convert the value to JSON
		encoded, err := json.Marshal(v)
		if err != nil {
			c.ResultNull()
		} else {
			c.ResultText(string(encoded))
		}

	}

}

// EOF is called after each row is queried to check if there are more rows
func (c *SQLiteCursor) EOF() bool {
	return c.noMoreRows && c.rows.Len() == 0
}

// Next is called to move the cursor to the next row
//
// If noMoreRows is set to false, and the cursor is at the end of the rows,
// Next will ask the plugin for more rows
//
// If noMoreRows is set to true, Next will set EOF to true
func (c *SQLiteCursor) Next() error {
	// Next is always called before scanning the row
	// Therefore, if there is one row left, it means we have already scanned it
	// and we must ask the plugin for more rows
	if c.rows.Len() <= 1 {
		// If the plugin stated that there are no more rows, we return
		if c.noMoreRows {
			c.rows.Clear()
			return nil
		}
		_, err := c.requestRowsFromPlugin()
		if err != nil {
			return err
		}
	}
	// We move the cursor to the next row
	c.rows.PopFront()

	return nil
}

// RowID is called to get the row ID of the current row
func (c *SQLiteCursor) Rowid() (int64, error) {
	// If the table has no primary key, we return a random number
	if c.schema.PrimaryKey == -1 {
		return rand.Int64(), nil
	}
	// Otherwise, we find the column that is the primary key
	// and return its value
	// TODO: handle the case where the primary key is a string
	columnID := c.schema.PrimaryKey
	id, ok := c.rows.Front()[columnID].(int64)
	if !ok {
		return 0, errors.New("could not convert the primary key to int64")
	}
	return id, nil
}

func (c *SQLiteCursor) Filter(idxNum int, idxStr string, vals []interface{}) error {
	// Filter can be called several times with the same cursor
	// Each time, it is supposed to reset the cursor to the beginning
	// Therefore, it should wipe out all the cursor fields
	//
	// Moreover, for the sake of simplicity, we will create a new cursor on the plugin side,
	// which means the cursorIndex must be incremented while not yelding any conflict
	// This is why we store a reference to the nextCursor field of the table

	// Reset the cursor to its initial state
	resetCursor(c)

	// We unmarshal the constraints from the IdxStr field
	// and store them in the constraints field of the cursor
	var err error
	err = loadConstraintsFromJSON(idxStr, &c.constraints, vals)
	if err != nil {
		return errors.Join(errors.New("could not load the constraints"), err)
	}

	// We request the rows from the plugin
	_, err = c.requestRowsFromPlugin()
	if err != nil {
		return err
	}

	return nil
}

const maxRowsFetchingRetry = 16

// requestRowsFromPlugin requests more rows to the plugin
//
// It returns the number of rows returned
func (cursor *SQLiteCursor) requestRowsFromPlugin() (int, error) {
	if cursor.noMoreRows {
		return 0, errors.New("requestRowsFromPlugin was called but plugin has no more rows")
	}

	// We request the rows from the plugin
	rows, noMoreRows, err := cursor.client.Plugin.Query(cursor.connectionIndex, cursor.tableIndex, cursor.cursorIndex, cursor.constraints)
	if err != nil {
		return 0, errors.Join(errors.New("could not request the rows from the plugin"), err)
	}
	// If the plugin did not return any rows, we retry
	i := 0
	for (!noMoreRows) && (len(rows) == 0 || rows == nil) && (i < maxRowsFetchingRetry) {
		rows, noMoreRows, err = cursor.client.Plugin.Query(cursor.connectionIndex, cursor.tableIndex, cursor.cursorIndex, cursor.constraints)
		i++
		time.Sleep(10 * time.Millisecond)
		if err != nil {
			return 0, errors.Join(errors.New("could not request the rows from the plugin"), err)
		}
	}
	if i == maxRowsFetchingRetry {
		return 0, errors.New("could not fetch any row from the plugin. Max retries reached")
	}
	// If the plugin stated that there are no more rows, we set noMoreRows to true
	cursor.noMoreRows = noMoreRows
	for _, row := range rows {
		cursor.rows.PushBack(row)
	}

	return len(rows), nil
}

// parseConstraintsFromSQLite parses the constraints from SQLite and stores them in the QueryConstraint struct
//
// For the offset and limit constraints, we store their position in the vals field
// so that we can pass them to the plugin
//
// For the IS NULL, IS, IS NOT NULL and IS NOT operators, we convert them to the EQUAL and NOT EQUAL operators
// because
func parseConstraintsFromSQLite(cst []sqlite3.InfoConstraint, ob []sqlite3.InfoOrderBy, constraints *rpc.QueryConstraint, used []bool, schema rpc.DatabaseSchema) {
	/*
		Internal notes:
		- The usable constraints are the ones that are used in the query
		- Any IS NULL, IS, IS NOT NULL and IS NOT operators are converted to EQUAL and NOT EQUAL operators
		- For the LIMIT and OFFSET constraints, we store their position in the vals field
		  and let the loader get the values
		- -1 as a value means SQL NULL. The loader will convert it to nil
		- nil as a value means we don't know the value yet. The loader will get it from the vals field

		I know it looks like a mess, will probably refactor it later
		But you know, nothing is more permanent than a temporary solution.
	*/

	constraints.Columns = make([]rpc.ColumnConstraint, 0, len(cst))

	// We iterate over the constraints and store the usable ones
	var tempOp rpc.Operator
	j := 0 // Keep track of the number of constraints used (for marking the LIMIT and OFFSET cols)
	for i, c := range cst {
		if c.Usable {
			tempOp = convertSQLiteOPtoOperator(c.Op)
			switch tempOp {
			case rpc.OperatorLimit:
				// We note the position of the LIMIT constraint in vals
				constraints.Limit = j
			case rpc.OperatorOffset:
				// We note the position of the OFFSET constraint in vals
				constraints.Offset = j
				// We check if the schema handles the OFFSET constraint
				// If not, we don't include it in vals
				// Furthermore, it will tell SQLite that it must handle the OFFSET itself
				// See https://github.com/julien040/go-sqlite3-anyquery/commit/f32fe2011fdf482c1a3c2f3c15dc85fb0e965550
				if !schema.HandleOffset {
					used[i] = false
				}
			// In all the other cases, we don't know the value yet
			// so we store the constraint as is
			default:
				constraints.Columns = append(constraints.Columns, rpc.ColumnConstraint{
					ColumnID: c.Column, // The column index
					Operator: tempOp,   // We convert the SQLite operator to our own operator
					Value:    nil,      // We don't know the value yet
				})
			}
			used[i] = true
			j++
		}
	}

	// We iterate over the ORDER BY constraints and store them in the OrderBy field
	constraints.OrderBy = make([]rpc.OrderConstraint, 0, len(ob))
	for _, o := range ob {
		constraints.OrderBy = append(constraints.OrderBy, rpc.OrderConstraint{
			ColumnID:   o.Column,
			Descending: o.Desc,
		})
	}
}

// convertSQLiteOPtoOperator converts a SQLite operator to an Operator
// known by anyquery
func convertSQLiteOPtoOperator(op sqlite3.Op) rpc.Operator {
	converted := int8(op)
	// Try to convert the operator
	opConverted := rpc.Operator(converted)
	return opConverted
}

// loadConstraintsFromJSON unmarshals the JSON serialized constraints
// from the IdxStr field of the IndexResult
// and stores them in the constraints field of the cursor
//
// It also infer the type of the value and stores it in the constraints field
func loadConstraintsFromJSON(idxStr string, constraints *rpc.QueryConstraint, vals []interface{}) error {
	err := json.Unmarshal([]byte(idxStr), &constraints)
	if err != nil {
		return errors.Join(errors.New("could not unmarshal the constraints"), err)
	}
	// We load the values from the vals field in the QueryConstraint struct

	// Fill the offset and limit constraints
	if constraints.Limit != -1 {
		constraints.Limit = int(vals[constraints.Limit].(int64))
	}
	if constraints.Offset != -1 {
		constraints.Offset = int(vals[constraints.Offset].(int64))
	}

	// J is the indice of the value in the vals field
	// We keep it separate from the loop because we need to increment it only when the value is not nil
	j := 0
	for i, cst := range constraints.Columns {
		switch cst.Operator {
		case rpc.OperatorLike:
			// We convert the LIKE string to a MATCH string
			// and store it in the constraints field
			constraints.Columns[i].Value = convertLikeToGlobString(vals[j].(string))
			constraints.Columns[i].Operator = rpc.OperatorGlob
			j++

		default:
			// If the value is -1, it means SQL NULL
			// so we fill it with nil
			// In the other cases, we fill it with the value in vals
			if constraints.Columns[i].Value == nil {
				constraints.Columns[i].Value = vals[j]
				j++
			} else {
				constraints.Columns[i].Value = nil
			}
		}

	}
	return nil
}

// convertLikeToGlobString converts a LIKE string to a MATCH string
//
// LIKE follows the SQL syntax with % and _
//
//	MATCH follows the UNIX glob syntax with * and ?
func convertLikeToGlobString(s string) string {
	// We replace the % with *
	// and the _ with ?
	// We also escape the * and ? with a backslash
	// to avoid any conflict
	return strings.ReplaceAll(strings.ReplaceAll(s, "%", "*"), "_", "?")
}

// resetCursor resets the cursor to its initial state
//
// It's useful when SQLite reuses the cursor
func resetCursor(c *SQLiteCursor) {
	c.noMoreRows = false
	c.rows.Clear()
	c.cursorIndex = *c.nextCursor
	*c.nextCursor++

	c.constraints = rpc.QueryConstraint{
		Limit:  -1,
		Offset: -1,
	}
}
