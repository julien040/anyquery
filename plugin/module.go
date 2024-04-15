package plugin

import (
	"encoding/json"
	"errors"
	rand "math/rand/v2"
	"reflect"
	"strings"
	"time"

	"github.com/gammazero/deque"
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
	PluginPath     string
	PluginManifest PluginManifest
	TableIndex     int
	client         *InternalClient
	UserConfig     map[string]string
}

// SQLiteTable that holds the information needed for the BestIndex and Open methods
type SQLiteTable struct {
	nextCursor int
	tableIndex int
	schema     DatabaseSchema
	client     *InternalClient
}

// SQLiteCursor holds the information needed for the Column, Filter, EOF and Next methods
type SQLiteCursor struct {
	tableIndex  int
	cursorIndex int
	schema      DatabaseSchema
	client      *InternalClient
	noMoreRows  bool
	rows        *deque.Deque[[]interface{}] // A ring buffer to store the rows before sending them to SQLite
	nextCursor  *int
	constraints QueryConstraint
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
	rpcClient, err := NewClient(m.PluginPath)
	if err != nil {
		return nil, errors.Join(errors.New("could not create a new rpc client for "+m.PluginPath), err)
	}
	m.client = rpcClient

	// Request the schema of the table from the plugin
	dbSchema, err := m.client.Plugin.Initialize(m.TableIndex, m.UserConfig)
	if err != nil {
		return nil, errors.Join(errors.New("could not request the schema of the table from the plugin "+m.PluginPath), err)
	}

	// Initialize a new table
	table := &SQLiteTable{
		0,
		m.TableIndex,
		dbSchema,
		m.client,
	}

	return table, nil
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
		if col.IsParameter && !present[i] {
			return nil, sqlite3.ErrConstraint
		}
	}

	// We serialize the constraints so that we can pass them to the Filter method
	// The only way to communicate them to the Filter method is through the IdxStr field
	// Therefore, we must serialize them as JSON and unmarshal them in the Filter method
	constraints := QueryConstraint{
		Limit:  -1,
		Offset: -1,
	}

	// Used is a boolean array that tells SQLite which constraints are used
	// and that must be passed to the Filter method in the vals field
	used := make([]bool, len(cst))
	parseConstraintsFromSQLite(cst, constraints, used, t.schema)

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
	cursor := &SQLiteCursor{
		t.tableIndex,
		t.nextCursor,
		t.schema,
		t.client,
		false,
		deque.New[[]interface{}](preAllocatedCapacity, minimumCapacityRingBuffer),
		&t.nextCursor,
		QueryConstraint{},
	}
	// We increment the cursor id for the next cursor by 1
	// so that the next cursor will have a different id
	t.nextCursor++

	return cursor, nil
}

// Close is called when the cursor is no longer needed
func (c *SQLiteCursor) Close() error { return nil }

// These methods are not used in this plugin
func (v *SQLiteTable) Disconnect() error {
	// We close the client
	v.client.Client.Kill()
	return nil
}
func (v *SQLiteTable) Destroy() error  { return nil }
func (v *SQLiteModule) DestroyModule() {}

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
		// Otherwise, we return the value of the column from the ring buffer
		if len(c.rows.Front()) <= col {
			// The plugin did not return enough columns. We return NULL
			// TODO: Must return a log message
			context.ResultNull()
		} else {
			convertToSQLiteVal(c.rows.Front()[col], context)
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
	case int, int8, int16, int32:
		c.ResultInt(int(reflect.ValueOf(v).Int()))
	case uint, uint8, uint16, uint32:
		c.ResultInt(int(reflect.ValueOf(v).Uint()))
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
	case nil:
		c.ResultNull()
	default:
		// TODO: Must return a log message
		c.ResultNull()
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
	// If the cursor is at the end of the rows
	// we ask the plugin for more rows
	if c.rows.Len() == 0 {
		// If the plugin stated that there are no more rows, we return
		if c.noMoreRows {
			return nil
		}
		_, err := c.requestRowsFromPlugin()
		if err != nil {
			return errors.Join(errors.New("could not request the rows from the plugin"), err)
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
	// How to fix this? We must have access to the parent struct (SQLiteTable).

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
		return errors.Join(errors.New("could not request the rows from the plugin"), err)
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
	rows, noMoreRows, err := cursor.client.Plugin.Query(cursor.tableIndex, cursor.cursorIndex, cursor.constraints)
	if err != nil {
		return 0, errors.Join(errors.New("could not request the rows from the plugin"), err)
	}
	i := 0
	for (!noMoreRows) && (len(rows) == 0) && (i < maxRowsFetchingRetry) {
		rows, noMoreRows, err = cursor.client.Plugin.Query(cursor.tableIndex, cursor.cursorIndex, cursor.constraints)
		time.Sleep(10 * time.Millisecond)
		if err != nil {
			return 0, errors.Join(errors.New("could not request the rows from the plugin"), err)
		}
	}
	if i == maxRowsFetchingRetry {
		return 0, errors.New("could not fetch any row from the plugin. Max retries reached")
	}
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
func parseConstraintsFromSQLite(cst []sqlite3.InfoConstraint, constraints QueryConstraint, used []bool, schema DatabaseSchema) {
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

	// We iterate over the constraints and store the usable ones
	var tempOp Operator
	j := 0 // Keep track of the number of constraints used (for marking the LIMIT and OFFSET cols)
	for i, c := range cst {
		if c.Usable {
			tempOp = convertSQLiteOPtoOperator(c.Op)
			switch tempOp {
			case OperatorLimit:
				// We note the position of the LIMIT constraint in vals
				constraints.Limit = j
			case OperatorOffset:
				// We note the position of the OFFSET constraint in vals
				constraints.Offset = j
				// We check if the schema handles the OFFSET constraint
				// If not, we don't include it in vals
				// Furthermore, it will tell SQLite that it must handle the OFFSET itself
				// See https://github.com/julien040/go-sqlite3-anyquery/commit/f32fe2011fdf482c1a3c2f3c15dc85fb0e965550
				if !schema.HandleOffset {
					used[i] = false
				}
			case OperatorIsNull:
				// We convert the IS NULL operator to the EQUAL operator
				constraints.Columns = append(constraints.Columns, ColumnConstraint{
					ColumnID: c.Column, // The column index
					Operator: OperatorEqual,
					Value:    -1, // -1 means SQL NULL | the loader will convert it to nil
				})
				continue // To avoid setting used[i] to true
			case OperatorIs:
				// We convert the IS operator to the EQUAL operator
				constraints.Columns = append(constraints.Columns, ColumnConstraint{
					ColumnID: c.Column, // The column index
					Operator: OperatorEqual,
					Value:    nil, // We don't know the value yet
				})
			case OperatorIsNotNull:
				// We convert the IS NOT NULL operator to the NOT EQUAL operator
				constraints.Columns = append(constraints.Columns, ColumnConstraint{
					ColumnID: c.Column, // The column index
					Operator: OperatorNotEqual,
					Value:    -1, // -1 means SQL NULL | the loader will convert it to nilt
				})
				continue
			case OperatorIsNot:
				// We convert the IS NOT operator to the NOT EQUAL operator
				constraints.Columns = append(constraints.Columns, ColumnConstraint{
					ColumnID: c.Column, // The column index
					Operator: OperatorNotEqual,
					Value:    nil, // We don't know the value yet
				})

			// In all the other cases, we don't know the value yet
			// so we store the constraint as is
			default:
				constraints.Columns = append(constraints.Columns, ColumnConstraint{
					ColumnID: c.Column, // The column index
					Operator: tempOp,   // We convert the SQLite operator to our own operator
					Value:    nil,      // We don't know the value yet
				})
			}
			used[i] = true
			j++
		}
	}
}

// loadConstraintsFromJSON unmashals the JSON serialized constraints
// from the IdxStr field of the IndexResult
// and stores them in the constraints field of the cursor
//
// It also infer the type of the value and stores it in the constraints field
func loadConstraintsFromJSON(idxStr string, constraints *QueryConstraint, vals []interface{}) error {
	err := json.Unmarshal([]byte(idxStr), &constraints)
	if err != nil {
		return errors.Join(errors.New("could not unmarshal the constraints"), err)
	}
	// We load the values from the vals field in the QueryConstraint struct

	// J is the indice of the value in the vals field
	// We keep it separate from the loop because we need to increment it only when the value is not nil
	j := 0
	for i, cst := range constraints.Columns {
		switch cst.Operator {
		case OperatorLike:
			// We convert the LIKE string to a MATCH string
			// and store it in the constraints field
			constraints.Columns[i].Value = convertLikeToMatchString(vals[j].(string))
			constraints.Columns[i].Operator = OperatorMatch
			j++

		default:
			// If the value is -1, it means SQL NULL
			// so we fill it with nil
			// In the other cases, we fill it with the value in vals
			if constraints.Columns[i].Value == nil {
				constraints.Columns[i].Value = vals[i]
				j++
			} else {
				constraints.Columns[i].Value = nil
			}
		}

	}
	return nil
}

// convertLikeToMatchString converts a LIKE string to a MATCH string
//
// LIKE follows the SQL syntax with % and _
//
//	MATCH follows the UNIX glob syntax with * and ?
func convertLikeToMatchString(s string) string {
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

	c.constraints = QueryConstraint{
		Limit:  -1,
		Offset: -1,
	}
}
