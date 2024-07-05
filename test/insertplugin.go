package main

import (
	"github.com/julien040/anyquery/rpc"
)

// This plugin registers two tables with the same table function

type myPlugin2 struct {
	rpc.Plugin
}

type table2 struct {
	rows [][]interface{}
}

type myReader2 struct {
	rows [][]interface{}
}

func (r *myReader2) Query(constraint rpc.QueryConstraint) ([][]interface{}, bool, error) {
	return r.rows, true, nil
}

func (t *table2) CreateReader() rpc.ReaderInterface {
	return &myReader2{
		rows: t.rows,
	}
}

func (t *table2) Insert(row [][]interface{}) error {
	for _, r := range row {
		t.rows = append(t.rows, r)
	}
	return nil
}

func (t *table2) Update(row [][]interface{}) error {
	for _, newRow := range row {
		for i, row := range t.rows {
			var actualPK int64
			switch row[0].(type) {
			case int:
				actualPK = int64(row[0].(int))
			case int64:
				actualPK = row[0].(int64)
			}

			if actualPK == newRow[0] {
				t.rows[i] = newRow[1:] // 1: because the first element is the primary key
				break
			}
		}
	}
	return nil
}

func (t *table2) Delete(row []interface{}) error {
	for i, r := range t.rows {
		var actualPK int64
		switch r[0].(type) {
		case int:
			actualPK = int64(r[0].(int))
		case int64:
			actualPK = r[0].(int64)
		}
		if actualPK == row[0] {
			t.rows = append(t.rows[:i], t.rows[i+1:]...)
		}
	}
	return nil
}

func (t *table2) Close() error {
	return nil
}

func main() {
	plugin := rpc.NewPlugin()

	rows := make([][]interface{}, 2, 128)
	rows[0] = []interface{}{1, "Alice", 20, "1234 Main St", 0}
	rows[1] = []interface{}{2, "Bob", 30, "5678 Elm St", 0}

	tableFunc := func(args rpc.TableCreatorArgs) (rpc.Table, *rpc.DatabaseSchema, error) {
		table := &table2{
			rows: rows,
		}
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
				{
					Name:        "age",
					Type:        rpc.ColumnTypeInt,
					IsParameter: false,
				},
				{
					Name:        "address",
					Type:        rpc.ColumnTypeString,
					IsParameter: false,
				},
				{
					Name:        "partition_id",
					Type:        rpc.ColumnTypeInt,
					IsParameter: true,
				},
			},
			PrimaryKey:    0,
			HandleOffset:  false,
			HandlesInsert: true,
			HandlesUpdate: true,
			HandlesDelete: true,
		}, nil
	}
	plugin.RegisterTable(0, tableFunc)

	plugin.RegisterTable(1, tableFunc)

	plugin.Serve()

}
