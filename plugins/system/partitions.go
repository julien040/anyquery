package main

import (
	"fmt"

	"github.com/julien040/anyquery/rpc"
	"github.com/shirou/gopsutil/v4/disk"
)

// A constructor to create a new table instance
// This function is called everytime a new connection is made to the plugin
//
// It should return a new table instance, the database schema and if there is an error
func partitionsCreator(args rpc.TableCreatorArgs) (rpc.Table, *rpc.DatabaseSchema, error) {
	return &partitionsTable{}, &rpc.DatabaseSchema{
		Columns: []rpc.DatabaseSchemaColumn{
			{
				Name: "device",
				Type: rpc.ColumnTypeString,
			},
			{
				Name: "mountpoint",
				Type: rpc.ColumnTypeString,
			},
			{
				Name: "fstype",
				Type: rpc.ColumnTypeString,
			},
			{
				Name: "options",
				Type: rpc.ColumnTypeString,
			},
		},
	}, nil
}

// The table struct
// There is one per connection to the plugin and is created by the creator function
// In there, you can store any state you need to read the rows (e.g. a database connection, an API token, etc.)
type partitionsTable struct {
}

// The cursor struct
// There is one per query and is created by the createReader function
// In there, you can store any state you need to read the rows (e.g. a database connection from partitionsTable, an offset, a cursor, etc.)
type partitionsCursor struct {
}

// Create a new cursor that will be used to read rows
func (t *partitionsTable) CreateReader() rpc.ReaderInterface {
	return &partitionsCursor{}
}

// Return a slice of rows that will be returned to Anyquery and filtered.
// The second return value is true if the cursor has no more rows to return
//
// The constraints are used for optimization purposes to "pre-filter" the rows
// If the rows returned don't match the constraints, it's not an issue. Anyquery will filter them out
func (t *partitionsCursor) Query(constraints rpc.QueryConstraint) ([][]interface{}, bool, error) {
	partitions, err := disk.Partitions(true)
	if err != nil {
		return nil, true, fmt.Errorf("failed to get partitions: %w", err)
	}

	rows := make([][]interface{}, 0, len(partitions))
	for _, partition := range partitions {
		rows = append(rows, []interface{}{
			partition.Device,
			partition.Mountpoint,
			partition.Fstype,
			partition.Opts,
		})
	}

	return rows, true, nil
}

// A slice of rows to insert
func (t *partitionsTable) Insert(rows [][]interface{}) error {
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
func (t *partitionsTable) Update(rows [][]interface{}) error {
	return nil
}

// A slice of primary keys to delete
func (t *partitionsTable) Delete(primaryKeys []interface{}) error {
	return nil
}

// A destructor to clean up resources
func (t *partitionsTable) Close() error {
	return nil
}
