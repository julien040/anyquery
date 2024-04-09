package plugin

import (
	"errors"

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
	// The first task of BestIndex is to check if the constraints are valid
	// If not, we return sqlite3.ErrConstraint
	return nil, nil
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
	}
	// We increment the cursor id for the next cursor by 1
	// so that the next cursor will have a different id
	t.nextCursor++

	return cursor, nil
}

// Close is called when the cursor is no longer needed
func (c *SQLiteCursor) Close() error { return nil }

// These methods are not used in this plugin
func (v *SQLiteTable) Disconnect() error { return nil }
func (v *SQLiteTable) Destroy() error    { return nil }
func (v *SQLiteModule) DestroyModule()   {}

// Column is called when a column is queried
//
// It should return the value of the column
func (c *SQLiteCursor) Column(context *sqlite3.SQLiteContext, col int) error { return nil }

// EOF is called after each row is queried to check if there are more rows
func (c *SQLiteCursor) EOF() bool { return false }

// Next is called to move the cursor to the next row
//
// If noMoreRows is set to false, and the cursor is at the end of the rows,
// Next will ask the plugin for more rows
//
// If noMoreRows is set to true, Next will set EOF to true
func (c *SQLiteCursor) Next() error { return nil }

// RowID is called to get the row ID of the current row
func (c *SQLiteCursor) Rowid() (int64, error) { return 0, nil }

func (c *SQLiteCursor) Filter(idxNum int, idxStr string, vals []interface{}) error {
	// Filter can be called several times with the same cursor
	// Each time, it is supposed to reset the cursor to the beginning
	// Therefore, it should wipe out all the cursor fields
	//
	// Moreover, for the sake of simplicity, we will create a new cursor on the plugin side,
	// which means the cursorIndex must be incremented while not yelding any conflict
	// How to fix this? We must have access to the parent struct (SQLiteTable).
	return nil
}
