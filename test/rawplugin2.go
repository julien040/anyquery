package main

import (
	go_plugin "github.com/hashicorp/go-plugin"
	"github.com/julien040/anyquery/plugin"
	"log"
)

type testPlugin struct {
	counter         int
	lastCursorIndex int
}

func (i *testPlugin) Initialize(tableIndex int, config plugin.PluginConfig) (plugin.DatabaseSchema, error) {

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
				IsParameter: false,
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
		},
		PrimaryKey:   -1,
		HandleOffset: true,
	}, nil
}

func (i *testPlugin) Query(tableIndex int, cursorIndex int, constraint plugin.QueryConstraint) ([][]interface{}, bool, error) {
	// When we have a new cursor, we reset the counter
	if cursorIndex != i.lastCursorIndex {
		i.counter = 0
		i.lastCursorIndex = cursorIndex
	}

	// We send an empty array the first time
	// to let anyquery retry the query
	if i.counter < 1 {
		log.Println("Plugin Query: Empty array to retry the query")
		i.counter++
		return [][]interface{}{}, false, nil
	}

	// This is to simulate no more rows. We send 6 rows once
	i.counter++

	var offset int
	if constraint.Offset == -1 {
		offset = 0
	} else {
		offset = constraint.Offset
	}

	log.Println("Offset", constraint.Offset)
	log.Println("Limit", constraint.Limit)

	// We convert to esoteric types to test the conversion
	return [][]interface{}{

		{i.counter * 10, "Franck", 3.14, true},
		{i.counter * 100, "Franck", float32(6.28), uint8(0)},
		{i.counter * 1000, "Julien", float64(3.14), int64(1)},
		{uint16(i.counter * 10000), "Julien", 6.28, false},
		{int32(i.counter * 100000)}, // This row will be filled with

	}[offset:], i.counter > 0, nil
}

func main() {
	go_plugin.Serve(&go_plugin.ServeConfig{
		HandshakeConfig: go_plugin.HandshakeConfig{
			ProtocolVersion:  plugin.ProtocolVersion,
			MagicCookieKey:   plugin.MagicCookieKey,
			MagicCookieValue: plugin.MagicCookieValue,
		},
		Plugins: map[string]go_plugin.Plugin{
			"plugin": &plugin.Plugin{Impl: &testPlugin{counter: 0, lastCursorIndex: -1}},
		},
	})

}
