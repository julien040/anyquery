package rpc

import (
	"errors"
	"fmt"
	"os"
	"syscall"
	"time"

	rpcPlugin "github.com/hashicorp/go-plugin"
)

// This global variable (I know it's bad) is used to check if the plugin is served
//
// We need it because at most one plugin can be served during the lifetime of the program
var pluginServed = false

// Args passed to the TableCreator function
//
// Implementation details: args are passed as a struct
// so that if we add more arguments in the future, old plugins will still work
type TableCreatorArgs struct {
	// UserConfig is the configuration passed by the user
	// during the configuration of the plugin profile
	UserConfig PluginConfig

	// TableIndex is the index of the table in the manifest (0-based)
	TableIndex int

	// ConnectionID is the index of the connection.
	// It is used to identify the connection in the plugin and can change between restarts
	ConnectionID int
}

// TableCreator is a function that creates a new table interface
// and returns the schema of the table
type TableCreator func(args TableCreatorArgs) (Table, *DatabaseSchema, error)

// Represents a table in the plugin
//
// If your table doesn't support insert, update or delete, you should specify it in the schema
// and the according methods will not be called.
// They can return an error or nil
type Table interface {
	// CreateReader must return a new instance of a table reader
	// A table can have several concurrent readers for better performance
	CreateReader() ReaderInterface

	// Close is called when the connection is closed
	//
	// It is used to free resources and close connections
	Close() error
}

type TableInsert interface {
	// Insert is called when the main program wants to insert rows
	//
	// The rows are passed as a 2D slice of interface{} where each row is a slice
	// and each element in the row is an interface{} representing the value.
	//
	// interface{} can be an int, string, int64, float64, []byte or nil
	Insert(rows [][]interface{}) error

	Table
}

type TableUpdate interface {
	// Update is called when the main program wants to update rows
	//
	// The rows are passed as a 2D slice of interface{} where each row is a slice
	// and each element in the row is an interface{} representing the value.
	//
	// The primary key is at the index specified in the schema and is also at the first index of the row.
	// It is used to update the value of the primary key in the row. The first value is the former value
	// and the second value is the new value.
	//
	// interface{} can be an int, string, int64, float64, []byte or nil
	Update(rows [][]interface{}) error

	Table
}

type TableDelete interface {
	// Delete is called when the main program wants to delete rows
	//
	// The primary keys are passed as an array of interface{}
	Delete(primaryKeys []interface{}) error
}

// ReaderInterface is an interface that must be implemented by the plugin
//
// It maps the methods required by anyquery to work
type ReaderInterface interface {

	// Query is a method that returns rows for a given SELECT query
	//
	// Constraints are passed as arguments for optimization purposes
	// However, the plugin is free to ignore them because
	// the main program will filter the results to match the constraints
	//
	// The first return value is a 2D slice of interface{} where each row is a slice
	// and each element in the row is an interface{} representing the value.
	// The second return value is a boolean that specifies whether the cursor is exhausted
	// The order and type of the values should match the schema of the table
	Query(constraint QueryConstraint) ([][]interface{}, bool, error)
}

// CursorKey is a struct used a key in a map to store the cursors
// of a table
type cursorKey struct {
	connectionIndex int
	tableIndex      int
	cursorIndex     int
}

// TableKey is a struct used a key in a map to store the tables
// of a connection
type tableKey struct {
	connectionIndex int
	tableIndex      int
}

// Plugin represents a plugin that can be loaded by anyquery
type Plugin struct {
	// table is a map that stores the table interfaces creators
	table map[int]TableCreator
	// cursors is a map that stores the readers of the tables
	cursors map[cursorKey]ReaderInterface
	// tableConnection maps a connection/tableID to a table interface
	tableConnection   map[tableKey]Table
	connectionStarted bool
}

// NewPlugin creates a new plugin
func NewPlugin(tables ...TableCreator) *Plugin {
	p := &Plugin{
		table:           make(map[int]TableCreator),
		cursors:         make(map[cursorKey]ReaderInterface),
		tableConnection: make(map[tableKey]Table),
	}
	for i, table := range tables {
		p.table[i] = table
	}
	return p
}

// RegisterTable registers a new table to the plugin
//
// The tableIndex must be unique and match the index in the manifest
func (p *Plugin) RegisterTable(tableIndex int, tableCreator TableCreator) error {
	if pluginServed {
		return fmt.Errorf("plugin is already served. It's impossible to register two or more plugins")
	}

	if p.table == nil {
		p.table = make(map[int]TableCreator)
	}

	if p.cursors == nil {
		p.cursors = make(map[cursorKey]ReaderInterface)
	}

	if p.tableConnection == nil {
		p.tableConnection = make(map[tableKey]Table)
	}

	if _, ok := p.table[tableIndex]; ok {
		return fmt.Errorf("table index is already registered")
	}
	p.table[tableIndex] = tableCreator

	return nil
}

// Serve is a method that starts the plugin
//
// once called, any attempt to modify the plugin will be rejected
func (p *Plugin) Serve() error {
	if p.connectionStarted {
		return fmt.Errorf("the plugin is already started")
	}
	if pluginServed {
		return fmt.Errorf("plugin is already served. It's impossible to serve two or more plugins")
	}
	pluginServed = true
	p.connectionStarted = true

	// Ensure no zombie process is left behind
	ticker := time.NewTicker(2 * time.Second)
	go func() {
		for {
			<-ticker.C
			// Check if the parent process is still alive
			// If not, we exit the plugin
			ppid := os.Getppid()
			if ppid == 1 {
				// Process exited and got reparented to init
				os.Exit(0)
			} else {
				// Check if the parent process is still alive
				// If not, we exit the plugin
				process, err := os.FindProcess(ppid)
				if err != nil {
					os.Exit(0)
				}

				// Send a signal to the parent process
				err = process.Signal(syscall.Signal(0))
				if err != nil {
					os.Exit(0)
				}
			}
		}
	}()

	internal := &internalInterface{plugin: p}

	rpcPlugin.Serve(&rpcPlugin.ServeConfig{
		Plugins: map[string]rpcPlugin.Plugin{
			"plugin": &InternalPlugin{Impl: internal},
		},
		HandshakeConfig: rpcPlugin.HandshakeConfig{
			ProtocolVersion:  ProtocolVersion,
			MagicCookieKey:   MagicCookieKey,
			MagicCookieValue: MagicCookieValue,
		},
	})

	return nil
}

// This struct implements the InternalExchangeInterface
// required by the plugin library to work
type internalInterface struct {
	plugin *Plugin
	InternalExchangeInterface
}

func (i *internalInterface) Initialize(connectionIndex int, tableIndex int, config PluginConfig) (schema DatabaseSchema, err error) {
	/* // We check if the table is registered
	_, ok := i.plugin.table[tableIndex]
	if !ok {
		return DatabaseSchema{}, fmt.Errorf("plugin did not register the table")
	}
	// We call the Initialize method of the table to fetch the schema
	// and return it to the main program
	schema, err := i.plugin.table[tableIndex].Initialize(config)
	return schema, err */

	// Catch the panic and return it as an error
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("plugin panicked while calling the table creator: %v", r)
		}
	}()

	// We create a new table and store it in the map
	// so that later, we can create new readers out of it

	funcToCall, ok := i.plugin.table[tableIndex]
	if !ok {
		return DatabaseSchema{}, fmt.Errorf("plugin did not register the table")
	}

	table, schemaA, err := funcToCall(TableCreatorArgs{
		UserConfig:   config,
		TableIndex:   tableIndex,
		ConnectionID: connectionIndex,
	})
	if err != nil {
		return DatabaseSchema{}, fmt.Errorf("plugin did not initialize the table. Error: %v", err)
	}
	i.plugin.tableConnection[tableKey{connectionIndex: connectionIndex, tableIndex: tableIndex}] = table

	// Set to true each flag (insert, update, delete) if the table implements the corresponding interface
	if _, ok := table.(TableInsert); ok {
		schemaA.HandlesInsert = true
	}

	if _, ok := table.(TableUpdate); ok {
		schemaA.HandlesUpdate = true
	}

	if _, ok := table.(TableDelete); ok {
		schemaA.HandlesDelete = true
	}

	return *schemaA, nil

}

func (i *internalInterface) Query(connectionIndex int, tableIndex int, cursorIndex int, constraint QueryConstraint) (rows [][]interface{}, noMoreRows bool, err error) {
	/* // We check if the table is registered
	_, ok := i.plugin.table[tableIndex]
	if !ok {
		return nil, false, fmt.Errorf("plugin did not register the table")
	}
	// We check if the cursor exists
	reader, ok := i.plugin.cursors[cursorKey{tableIndex: tableIndex, cursorIndex: cursorIndex}]
	if !ok {
		// We create a new cursor
		reader = i.plugin.table[tableIndex].CreateReader()
		i.plugin.cursors[cursorKey{tableIndex: tableIndex, cursorIndex: cursorIndex}] = reader
	}

	// We call the Query method of the reader to fetch the rows
	// and return them to the main program
	rows, noMoreRows, err := reader.Query(constraint)
	return rows, noMoreRows, err */

	// Catch the panic and return it as an error
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("plugin panicked while running Query: %v", r)
		}
	}()

	// We check if the table is registered
	_, ok := i.plugin.table[tableIndex]
	if !ok {
		return nil, false, fmt.Errorf("plugin did not register the table")
	}

	// We check if the table is registered
	table, ok := i.plugin.tableConnection[tableKey{connectionIndex: connectionIndex, tableIndex: tableIndex}]
	if !ok {
		return nil, false, fmt.Errorf("main program did not initialize the table before querying it")
	}

	// We check if the cursor exists
	cursor := cursorKey{connectionIndex: connectionIndex, tableIndex: tableIndex, cursorIndex: cursorIndex}
	reader, ok := i.plugin.cursors[cursor]
	if !ok {
		// We create a new cursor
		reader = table.CreateReader()
		i.plugin.cursors[cursor] = reader
	}

	// We call the Query method of the reader to fetch the rows
	// and return them to the main program
	rows, noMoreRows, err = reader.Query(constraint)
	return rows, noMoreRows, err
}

func (i *internalInterface) Insert(connectionIndex int, tableIndex int, rows [][]interface{}) (err error) {
	// We check if the table is registered
	_, ok := i.plugin.table[tableIndex]
	if !ok {
		return fmt.Errorf("plugin did not register the table")
	}

	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("plugin panicked while running Insert: %v", r)
		}
	}()

	// We check if the connection initialized the table
	table, ok := i.plugin.tableConnection[tableKey{connectionIndex: connectionIndex, tableIndex: tableIndex}]
	if !ok {
		return fmt.Errorf("main program did not initialize the table before inserting into it")
	}

	if tableInsert, ok := table.(TableInsert); !ok {
		return fmt.Errorf("plugin does not support Insert")
	} else {
		// We call the Insert method of the table to insert the rows only if it is implemented
		return tableInsert.Insert(rows)
	}

}

func (i *internalInterface) Update(connectionIndex int, tableIndex int, rows [][]interface{}) (err error) {
	// We check if the table is registered
	_, ok := i.plugin.table[tableIndex]
	if !ok {
		return fmt.Errorf("plugin did not register the table")
	}

	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("plugin panicked while running Update: %v", r)
		}
	}()

	// We check if the connection initialized the table
	table, ok := i.plugin.tableConnection[tableKey{connectionIndex: connectionIndex, tableIndex: tableIndex}]
	if !ok {
		return fmt.Errorf("main program did not initialize the table before updating it")
	}

	if tableUpdate, ok := table.(TableUpdate); !ok {
		return fmt.Errorf("plugin does not support Update")
	} else {
		// We call the Update method of the table to update the rows only if it is implemented
		return tableUpdate.Update(rows)
	}
}

func (i *internalInterface) Delete(connectionIndex int, tableIndex int, primaryKeys []interface{}) (err error) {
	// We check if the table is registered
	_, ok := i.plugin.table[tableIndex]
	if !ok {
		return fmt.Errorf("plugin did not register the table")
	}

	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("plugin panicked while running Delete: %v", r)
		}
	}()

	// We check if the connection initialized the table
	table, ok := i.plugin.tableConnection[tableKey{connectionIndex: connectionIndex, tableIndex: tableIndex}]
	if !ok {
		return fmt.Errorf("main program did not initialize the table before deleting from it")
	}

	if tableDelete, ok := table.(TableDelete); !ok {
		return fmt.Errorf("plugin does not support Delete")
	} else {
		// We call the Delete method of the table to delete the rows only if it is implemented
		return tableDelete.Delete(primaryKeys)
	}
}

func (i *internalInterface) Close(connectionIndex int) error {
	// For each table of the connection, we call the Close method
	var chainedError error = nil
	for key, table := range i.plugin.tableConnection {
		if key.connectionIndex == connectionIndex {
			err := table.Close()
			// If an error occurs, we chain it with the previous one
			// and still continue to close the other tables
			if err != nil {
				return errors.Join(chainedError, fmt.Errorf("error while closing the table %v: %v", key.tableIndex, err))
			}
		}
	}
	return chainedError
}
