package plugin

import (
	"fmt"

	rpcPlugin "github.com/hashicorp/go-plugin"
	"github.com/julien040/anyquery/rpc"
)

// This global variable (I know it's bad) is used to check if the plugin is served
//
// We need it because at most one plugin can be served during the lifetime of the program
var pluginServed = false

type QueryConstraint = rpc.QueryConstraint

type Table interface {
	// CreateReader must return a new instance of the reader
	CreateReader() ReaderInterface

	// Initialize is a method that is called when the plugin is initialized
	//
	// It is called once when the plugin is loaded and is used by the main
	// program to infer the schema of the tables
	Initialize(config rpc.PluginConfig) (rpc.DatabaseSchema, error)
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
	Query(constraint rpc.QueryConstraint) ([][]interface{}, bool, error)
}

// CursorKey is a struct used a key in a map to store the cursors
// of a table
type cursorKey struct {
	tableIndex  int
	cursorIndex int
}

// Plugin represents a plugin that can be loaded by anyquery
type Plugin struct {
	// Unexported fields
	table             map[int]Table
	cursors           map[cursorKey]ReaderInterface
	connectionStarted bool
}

// NewPlugin returns a new instance of the plugin
func NewPlugin() *Plugin {
	return &Plugin{
		table:   make(map[int]Table),
		cursors: make(map[cursorKey]ReaderInterface),
	}
}

func (p *Plugin) New(val map[int]Table) {
	// Copy the map
	for k, v := range val {
		p.table[k] = v
	}
	p.cursors = make(map[cursorKey]ReaderInterface)
}

// RegisterTable registers a new table to the plugin
//
// The tableIndex must be unique and match the index in the manifest
func (p *Plugin) RegisterTable(tableIndex int, table Table) error {
	if pluginServed {
		return fmt.Errorf("plugin is already served. It's impossible to register two or more plugins")

	}
	if _, ok := p.table[tableIndex]; ok {
		return fmt.Errorf("table index is already registered")
	}
	p.table[tableIndex] = table
	return nil
}

// GetTable returns the tables registered in the plugin
func (p *Plugin) GetTable() []int {
	var res []int
	for k := range p.table {
		res = append(res, k)
	}
	return res
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

	internal := &internalInterface{plugin: p}

	rpcPlugin.Serve(&rpcPlugin.ServeConfig{
		Plugins: map[string]rpcPlugin.Plugin{
			"plugin": &rpc.Plugin{Impl: internal},
		},
		HandshakeConfig: rpcPlugin.HandshakeConfig{
			ProtocolVersion:  rpc.ProtocolVersion,
			MagicCookieKey:   rpc.MagicCookieKey,
			MagicCookieValue: rpc.MagicCookieValue,
		},
	})

	return nil
}

// This struct implements the InternalExchangeInterface
// required by the plugin library to work
type internalInterface struct {
	plugin *Plugin
	rpc.InternalExchangeInterface
}

func (i *internalInterface) Initialize(tableIndex int, config rpc.PluginConfig) (rpc.DatabaseSchema, error) {
	// We check if the table is registered
	_, ok := i.plugin.table[tableIndex]
	if !ok {
		return rpc.DatabaseSchema{}, fmt.Errorf("plugin did not register the table")
	}
	// We call the Initialize method of the table to fetch the schema
	// and return it to the main program
	schema, err := i.plugin.table[tableIndex].Initialize(config)
	return schema, err
}

func (i *internalInterface) Query(tableIndex int, cursorIndex int, constraint rpc.QueryConstraint) ([][]interface{}, bool, error) {
	// We check if the table is registered
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
	return rows, noMoreRows, err
}
