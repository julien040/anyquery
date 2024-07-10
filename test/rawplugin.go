package main

import (
	go_plugin "github.com/hashicorp/go-plugin"
	plugin "github.com/julien040/anyquery/rpc"
)

type testPlugin struct {
	counter         int
	lastCursorIndex int
}

func (i *testPlugin) Initialize(connectionID int, tableIndex int, config plugin.PluginConfig) (plugin.DatabaseSchema, error) {
	return plugin.DatabaseSchema{
		Columns: []plugin.DatabaseSchemaColumn{
			{
				Name:        "id",
				Type:        plugin.ColumnTypeInt,
				IsParameter: false,
			},
			{
				Name:        "name",
				Type:        plugin.ColumnTypeString,
				IsParameter: true,
				IsRequired:  true,
			},
			{
				Name:        "size",
				Type:        plugin.ColumnTypeFloat,
				IsParameter: false,
			},
			{
				Name:        "is_active",
				Type:        plugin.ColumnTypeInt,
				IsParameter: false,
			},
			{
				Name:        "data",
				Type:        plugin.ColumnTypeBlob,
				IsParameter: false,
			},
		},
		PrimaryKey:   0,
		HandleOffset: false,
	}, nil
}

func (i *testPlugin) Query(connectionID int, tableIndex int, cursorIndex int, constraint plugin.QueryConstraint) ([][]interface{}, bool, error) {
	// When we have a new cursor, we reset the counter
	if cursorIndex != i.lastCursorIndex {
		i.counter = 0
		i.lastCursorIndex = cursorIndex
	}
	// This is to simulate no more rows. We send 4 rows 5 times
	i.counter++
	return [][]interface{}{

		{i.counter * 10, 3.14, true},
		{i.counter * 100, 6.28, false},
		{i.counter * 1000, 3.14, true},
		{i.counter * 10000, 6.28, false},
	}, (i.counter > 4), nil
}

func (i *testPlugin) Close(connectionID int) error {
	return nil
}

func (i *testPlugin) Insert(connectionID int, tableIndex int, rows [][]interface{}) error {
	return nil
}

func (i *testPlugin) Update(connectionID int, tableIndex int, rows [][]interface{}) error {
	return nil
}

func (i *testPlugin) Delete(connectionID int, tableIndex int, primaryKeys []interface{}) error {
	return nil
}

func main() {
	go_plugin.Serve(&go_plugin.ServeConfig{
		HandshakeConfig: go_plugin.HandshakeConfig{
			ProtocolVersion:  plugin.ProtocolVersion,
			MagicCookieKey:   plugin.MagicCookieKey,
			MagicCookieValue: plugin.MagicCookieValue,
		},
		Plugins: map[string]go_plugin.Plugin{
			"plugin": &plugin.InternalPlugin{Impl: &testPlugin{counter: 0, lastCursorIndex: -1}},
		},
	})

}
