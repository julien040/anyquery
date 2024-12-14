package main

import (
	"fmt"

	"github.com/julien040/anyquery/rpc"
	"github.com/shirou/gopsutil/v4/cpu"
)

// A constructor to create a new table instance
// This function is called everytime a new connection is made to the plugin
//
// It should return a new table instance, the database schema and if there is an error
func cpu_statsCreator(args rpc.TableCreatorArgs) (rpc.Table, *rpc.DatabaseSchema, error) {
	return &cpu_statsTable{}, &rpc.DatabaseSchema{
		Columns: []rpc.DatabaseSchemaColumn{
			{
				Name: "cpu",
				Type: rpc.ColumnTypeString,
			},
			{
				Name: "user",
				Type: rpc.ColumnTypeFloat,
			},
			{
				Name: "system",
				Type: rpc.ColumnTypeFloat,
			},
			{
				Name: "idle",
				Type: rpc.ColumnTypeFloat,
			},
			{
				Name: "nice",
				Type: rpc.ColumnTypeFloat,
			},
			{
				Name: "iowait",
				Type: rpc.ColumnTypeFloat,
			},
			{
				Name: "irq",
				Type: rpc.ColumnTypeFloat,
			},
			{
				Name: "softirq",
				Type: rpc.ColumnTypeFloat,
			},
			{
				Name: "steal",
				Type: rpc.ColumnTypeFloat,
			},
			{
				Name: "guest",
				Type: rpc.ColumnTypeFloat,
			},
			{
				Name: "guest_nice",
				Type: rpc.ColumnTypeFloat,
			},
		},
	}, nil
}

// The table struct
// There is one per connection to the plugin and is created by the creator function
// In there, you can store any state you need to read the rows (e.g. a database connection, an API token, etc.)
type cpu_statsTable struct {
}

// The cursor struct
// There is one per query and is created by the createReader function
// In there, you can store any state you need to read the rows (e.g. a database connection from cpu_statsTable, an offset, a cursor, etc.)
type cpu_statsCursor struct {
}

// Create a new cursor that will be used to read rows
func (t *cpu_statsTable) CreateReader() rpc.ReaderInterface {
	return &cpu_statsCursor{}
}

// Return a slice of rows that will be returned to Anyquery and filtered.
// The second return value is true if the cursor has no more rows to return
//
// The constraints are used for optimization purposes to "pre-filter" the rows
// If the rows returned don't match the constraints, it's not an issue. Anyquery will filter them out
func (t *cpu_statsCursor) Query(constraints rpc.QueryConstraint) ([][]interface{}, bool, error) {
	stats, err := cpu.Times(true)
	if err != nil {
		return nil, true, fmt.Errorf("could not get cpu stats: %w", err)
	}

	rows := make([][]interface{}, 0, len(stats))
	for _, stat := range stats {
		rows = append(rows, []interface{}{
			stat.CPU,
			stat.User,
			stat.System,
			stat.Idle,
			stat.Nice,
			stat.Iowait,
			stat.Irq,
			stat.Softirq,
			stat.Steal,
			stat.Guest,
			stat.GuestNice,
		})
	}

	return rows, true, nil
}

// A slice of rows to insert
func (t *cpu_statsTable) Insert(rows [][]interface{}) error {
	// Example: insert the rows in a database
	// for _, row := range rows {
	// 	err := db.Insert(row[0], row[1], row[2])
	// 	if err != nil {
	// 		return err
	// 	}
	return nil
}

// A slice of rows to update
// The first element of each row is the primary key
// while the rest are the values to update
// The primary key is therefore present twice
func (t *cpu_statsTable) Update(rows [][]interface{}) error {
	return nil
}

// A slice of primary keys to delete
func (t *cpu_statsTable) Delete(primaryKeys []interface{}) error {
	return nil
}

// A destructor to clean up resources
func (t *cpu_statsTable) Close() error {
	return nil
}
