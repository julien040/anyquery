// Package rpc implements functions and structs for plugins to communicate with the main program
//
// All exported elements prefixed with "Internal" are not meant to be used by plugins
// but by programs that communicate with plugins
//
// We strongly recommend looking at the introduction to plugins in the README.md
// before looking at this package
package rpc

// This file is mostly boilerplate code that is required to communicate with the main program
// thanks to hashicorp/go-plugin.
//
// To learn more about how this works, you can see examples at:
// https://github.com/hashicorp/go-plugin

import (
	"errors"
	"net/rpc"
	"os/exec"
	"sync"
	"sync/atomic"
	"time"

	"github.com/hashicorp/go-hclog"
	go_plugin "github.com/hashicorp/go-plugin"
)

const ProtocolVersion = 1
const MagicCookieKey = "ANYQUERY_PLUGIN"
const MagicCookieValue = "1.0.0"

type InternalPlugin struct {
	Impl InternalExchangeInterface
}

// PluginRPCClient is a struct that holds the RPC client
// that will be called from the main program
type PluginRPCClient struct {
	client *rpc.Client
}

// PluginRPCServer is a struct that holds the RPC server
// that will be runned by the plugin for the main program
type PluginRPCServer struct {
	Impl InternalExchangeInterface
}

func (p *InternalPlugin) Server(*go_plugin.MuxBroker) (interface{}, error) {
	return &PluginRPCServer{Impl: p.Impl}, nil
}

func (p *InternalPlugin) Client(b *go_plugin.MuxBroker, c *rpc.Client) (interface{}, error) {
	return &PluginRPCClient{client: c}, nil
}

// -- Implementation of the RPC methods --

// InitializeArgs is a struct that holds the arguments for the Initialize method
//
// It's necessary to define this struct because the arguments for the RPC methods
// are passed as a single argument
type InitializeArgs struct {
	ConnectionID int
	TableIndex   int
	Config       PluginConfig
}

// QueryArgs is a struct that holds the arguments for the Query method (see InitializeArgs)
type QueryArgs struct {
	ConnectionID int
	TableIndex   int
	CursorIndex  int
	Constraint   QueryConstraint
}

type QueryReturn struct {
	Rows       [][]interface{}
	NoMoreRows bool
}

type InsertArgs struct {
	ConnectionID int
	TableIndex   int
	Rows         [][]interface{}
}

type UpdateArgs struct {
	ConnectionID int
	TableIndex   int
	Rows         [][]interface{}
}

type DeleteArgs struct {
	ConnectionID int
	TableIndex   int
	PrimaryKeys  []interface{}
}

func (m *PluginRPCClient) Initialize(connectionID int, tableIndex int, config PluginConfig) (DatabaseSchema, error) {
	args := &InitializeArgs{
		ConnectionID: connectionID,
		TableIndex:   tableIndex,
		Config:       config,
	}
	var resp DatabaseSchema
	err := m.client.Call("Plugin.Initialize", args, &resp)
	return resp, err
}

func (m *PluginRPCClient) Query(connectionID int, tableIndex int, cursorIndex int, constraint QueryConstraint) ([][]interface{}, bool, error) {
	args := &QueryArgs{
		ConnectionID: connectionID,
		TableIndex:   tableIndex,
		CursorIndex:  cursorIndex,
		Constraint:   constraint,
	}
	var resp QueryReturn
	err := m.client.Call("Plugin.Query", args, &resp)
	return resp.Rows, resp.NoMoreRows, err
}

func (m *PluginRPCClient) Insert(connectionID int, tableIndex int, rows [][]interface{}) error {
	return m.client.Call("Plugin.Insert", &InsertArgs{ConnectionID: connectionID, TableIndex: tableIndex, Rows: rows}, nil)
}

func (m *PluginRPCClient) Update(connectionID int, tableIndex int, rows [][]interface{}) error {
	return m.client.Call("Plugin.Update", &UpdateArgs{ConnectionID: connectionID, TableIndex: tableIndex, Rows: rows}, nil)
}

func (m *PluginRPCClient) Delete(connectionID int, tableIndex int, primaryKeys []interface{}) error {
	return m.client.Call("Plugin.Delete", &DeleteArgs{ConnectionID: connectionID, TableIndex: tableIndex, PrimaryKeys: primaryKeys}, nil)
}

func (m *PluginRPCClient) Close(connectionID int) error {
	return m.client.Call("Plugin.Close", connectionID, nil)
}

func (m *PluginRPCServer) Initialize(args *InitializeArgs, resp *DatabaseSchema) error {
	var err error
	*resp, err = m.Impl.Initialize(args.ConnectionID, args.TableIndex, args.Config)
	return err
}

func (m *PluginRPCServer) Query(args *QueryArgs, resp *QueryReturn) error {
	var err error
	resp.Rows, resp.NoMoreRows, err = m.Impl.Query(args.ConnectionID, args.TableIndex, args.CursorIndex, args.Constraint)
	return err
}

func (m *PluginRPCServer) Insert(args *InsertArgs, resp *struct{}) error {
	return m.Impl.Insert(args.ConnectionID, args.TableIndex, args.Rows)
}

func (m *PluginRPCServer) Update(args *UpdateArgs, resp *struct{}) error {
	return m.Impl.Update(args.ConnectionID, args.TableIndex, args.Rows)
}

func (m *PluginRPCServer) Delete(args *DeleteArgs, resp *struct{}) error {
	return m.Impl.Delete(args.ConnectionID, args.TableIndex, args.PrimaryKeys)
}

func (m *PluginRPCServer) Close(connectionID int, resp *struct{}) error {
	return m.Impl.Close(connectionID)
}

// -- End of implementation of the RPC methods --

type InternalClient struct {
	Client *go_plugin.Client
	Plugin InternalExchangeInterface
}

// ConnectionPool is a struct that holds the connections to the plugins
// It allows using the same executable for multiple connections
//
// It is not intended to be used by plugins but by the main program
type ConnectionPool struct {
	connections map[string]*struct {
		client          *InternalClient
		connectionCount atomic.Int32
	}
	// To ensure we are not creating multiple connections at the same time
	mu sync.Mutex
}

// NewConnectionPool creates a new connection pool
//
// Using the zero value is not recommended and might lead to a SIGSEGV
func NewConnectionPool() *ConnectionPool {
	return &ConnectionPool{
		connections: make(map[string]*struct {
			client          *InternalClient
			connectionCount atomic.Int32
		}),

		mu: sync.Mutex{},
	}
}

// Request a new client from the connection pool. Each NewClient must be followed by a CloseConnection.
// If these requirements are not met, the executable will not be killed
func (c *ConnectionPool) NewClient(executableLocation string, logger hclog.Logger) (*InternalClient, error) {

	// We create a new client if it doesn't exist
	// To ensure we are not creating multiple connections at the same time
	// We use a mutex
	c.mu.Lock()
	defer c.mu.Unlock()

	// We check if the client already exists
	if client, ok := c.connections[executableLocation]; ok {
		client.connectionCount.Add(1)
		return client.client, nil
	}

	client := new(InternalClient)

	// We use the same magic cookie as the main program
	// to ensure that the plugin is compatible with the main program

	client.Client = go_plugin.NewClient(&go_plugin.ClientConfig{
		HandshakeConfig: go_plugin.HandshakeConfig{
			ProtocolVersion:  ProtocolVersion,
			MagicCookieKey:   MagicCookieKey,
			MagicCookieValue: MagicCookieValue,
		},
		Plugins: map[string]go_plugin.Plugin{
			"plugin": &InternalPlugin{},
		},
		Cmd:    exec.Command(executableLocation),
		Logger: logger,
	})

	// We get the RPC client
	protocol, err := client.Client.Client()
	if err != nil {
		return nil, err
	}

	// We request the plugin
	raw, err := protocol.Dispense("plugin")
	if err != nil {
		return nil, err
	}

	// We cast the plugin to the InternalExchangeInterface
	plugin, ok := raw.(InternalExchangeInterface)
	if !ok {
		return nil, errors.New("plugin does not implement InternalExchangeInterface")
	}

	client.Plugin = plugin

	// We add the client to the connection pool
	c.connections[executableLocation] = &struct {
		client          *InternalClient
		connectionCount atomic.Int32
	}{
		client:          client,
		connectionCount: atomic.Int32{},
	}

	// We increment the connection count
	c.connections[executableLocation].connectionCount.Add(1)

	return client, nil
}

// Warn the plugin that the connection will be closed, wait a few seconds and close the connection
// If all connections are closed, the plugin is killed
func (c *ConnectionPool) CloseConnection(executableLocation string, connectionID int) {
	if client, ok := c.connections[executableLocation]; ok {
		client.connectionCount.Add(-1)
		// Call the destructor for the connection with a timeout of 5 seconds
		timer := time.NewTimer(5 * time.Second)

		chanClose := make(chan struct{})
		go func() {
			client.client.Plugin.Close(connectionID)
			chanClose <- struct{}{}
			timer.Stop()
		}()

		// Wait for the first event to happen
		select {
		case <-timer.C:
			break
		case <-chanClose:
			break
		}

		// If there are no more connections, we kill the client
		// Those two functions can be safely called concurrently
		// - delete is a no-op if the key doesn't exist
		// - Kill is a no-op if the client is already killed
		//
		// So we don't need to lock the map
		if client.connectionCount.Load() <= 0 {
			client.client.Client.Kill()
			delete(c.connections, executableLocation)
		}
	}
}
