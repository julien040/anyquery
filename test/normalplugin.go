package main

import (
	"github.com/julien040/anyquery/plugin"
	"github.com/julien040/anyquery/rpc"
)

type myPlugin struct {
	plugin.Table
}

func (m *myPlugin) Initialize(config rpc.PluginConfig) (rpc.DatabaseSchema, error) {
	return rpc.DatabaseSchema{
		Columns: []rpc.DatabaseSchemaColumn{
			{
				Name:        "id",
				Type:        rpc.ColumnTypeInt,
				IsParameter: false,
			},
			{
				Name:        "name",
				Type:        rpc.ColumnTypeString,
				IsParameter: false,
			},
		},
		PrimaryKey:   -1,
		HandleOffset: false,
	}, nil
}

func (m *myPlugin) CreateReader() plugin.ReaderInterface {
	return &myReader{}
}

type myReader struct {
}

func (m *myReader) Query(constraint rpc.QueryConstraint) ([][]interface{}, bool, error) {
	return [][]interface{}{
		{1, "hello"},
		{2, "world"}}, true, nil
}

func main() {
	plugin := plugin.NewPlugin()
	plugin.RegisterTable(0, &myPlugin{})
	plugin.RegisterTable(1, &myPlugin{})

	plugin.Serve()

}
