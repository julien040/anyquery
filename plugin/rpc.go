package plugin

// This file is mostly boilerplate code that is required to communicate with the main program
// thanks to hashicorp/go-plugin.
//
// To learn more about how this works, you can see examples at:
// https://github.com/hashicorp/go-plugin

import (
	"net/rpc"
	"os/exec"

	"errors"

	go_plugin "github.com/hashicorp/go-plugin"
)

const ProtocolVersion = 1
const MagicCookieKey = "ANYQUERY_PLUGIN"
const MagicCookieValue = "1.0.0"

type Plugin struct {
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

func (p *Plugin) Server(*go_plugin.MuxBroker) (interface{}, error) {
	return &PluginRPCServer{Impl: p.Impl}, nil
}

func (p *Plugin) Client(b *go_plugin.MuxBroker, c *rpc.Client) (interface{}, error) {
	return &PluginRPCClient{client: c}, nil
}

// -- Implementation of the RPC methods --

// InitializeArgs is a struct that holds the arguments for the Initialize method
//
// It's necessary to define this struct because the arguments for the RPC methods
// are passed as a single argument
type InitializeArgs struct {
	TableIndex int
	Config     PluginConfig
}

// QueryArgs is a struct that holds the arguments for the Query method (see InitializeArgs)
type QueryArgs struct {
	TableIndex  int
	CursorIndex int
	Constraint  QueryConstraint
}

type QueryReturn struct {
	Rows       [][]interface{}
	NoMoreRows bool
}

func (m *PluginRPCClient) Initialize(tableIndex int, config PluginConfig) (DatabaseSchema, error) {
	args := &InitializeArgs{
		TableIndex: tableIndex,
		Config:     config,
	}
	var resp DatabaseSchema
	err := m.client.Call("Plugin.Initialize", args, &resp)
	return resp, err
}

func (m *PluginRPCClient) Query(tableIndex int, cursorIndex int, constraint QueryConstraint) ([][]interface{}, bool, error) {
	args := &QueryArgs{
		TableIndex:  tableIndex,
		CursorIndex: cursorIndex,
		Constraint:  constraint,
	}
	var resp QueryReturn
	err := m.client.Call("Plugin.Query", args, &resp)
	return resp.Rows, resp.NoMoreRows, err
}

func (m *PluginRPCServer) Initialize(args *InitializeArgs, resp *DatabaseSchema) error {
	var err error
	*resp, err = m.Impl.Initialize(args.TableIndex, args.Config)
	return err
}

func (m *PluginRPCServer) Query(args *QueryArgs, resp *QueryReturn) error {
	var err error
	resp.Rows, resp.NoMoreRows, err = m.Impl.Query(args.TableIndex, args.CursorIndex, args.Constraint)
	return err
}

// -- End of implementation of the RPC methods --

type InternalClient struct {
	Client *go_plugin.Client
	Plugin InternalExchangeInterface
}

func NewClient(executableLocation string) (*InternalClient, error) {
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
			"plugin": &Plugin{},
		},
		Cmd: exec.Command(executableLocation),
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

	return client, nil

}
