package main

import (
	"github.com/julien040/anyquery/rpc"
)

// This plugin registers two tables with the same table function

type myPlugin struct {
	rpc.Plugin
}

type table1 struct {
}

type myReader struct {
}

func (r *myReader) Query(constraint rpc.QueryConstraint) ([][]interface{}, bool, error) {
	return [][]interface{}{
		{1, "hello"},
		{2, "world"}}, true, nil
}

func (t *table1) CreateReader() rpc.ReaderInterface {
	return &myReader{}
}

func (t *table1) Close() error {
	return nil
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

func main() {
	plugin := rpc.NewPlugin()

	tableFunc := func(args rpc.TableCreatorArgs) (rpc.Table, *rpc.DatabaseSchema, error) {
		table := &table1{}
		return table, &rpc.DatabaseSchema{
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
	plugin.RegisterTable(0, tableFunc)

	plugin.RegisterTable(1, tableFunc)

	plugin.Serve()

}
