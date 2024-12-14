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
func partition_usageCreator(args rpc.TableCreatorArgs) (rpc.Table, *rpc.DatabaseSchema, error) {
	return &partition_usageTable{}, &rpc.DatabaseSchema{
		HandlesInsert: false,
		HandlesUpdate: false,
		HandlesDelete: false,
		HandleOffset:  false,
		Columns: []rpc.DatabaseSchemaColumn{
			{
				Name:        "mountpoint",
				Type:        rpc.ColumnTypeString,
				IsParameter: true,
				IsRequired:  true,
			},
			{
				Name: "fstype",
				Type: rpc.ColumnTypeString,
			},
			{
				Name: "total",
				Type: rpc.ColumnTypeInt,
			},
			{
				Name: "free",
				Type: rpc.ColumnTypeInt,
			},
			{
				Name: "used",
				Type: rpc.ColumnTypeInt,
			},
			{
				Name: "used_percent",
				Type: rpc.ColumnTypeFloat,
			},
			{
				Name: "inodes_total",
				Type: rpc.ColumnTypeInt,
			},
			{
				Name: "inodes_used",
				Type: rpc.ColumnTypeInt,
			},
			{
				Name: "inodes_free",
				Type: rpc.ColumnTypeInt,
			},
			{
				Name: "inodes_used_percent",
				Type: rpc.ColumnTypeFloat,
			},
		},
	}, nil
}

// The table struct
// There is one per connection to the plugin and is created by the creator function
// In there, you can store any state you need to read the rows (e.g. a database connection, an API token, etc.)
type partition_usageTable struct {
}

// The cursor struct
// There is one per query and is created by the createReader function
// In there, you can store any state you need to read the rows (e.g. a database connection from partition_usageTable, an offset, a cursor, etc.)
type partition_usageCursor struct {
}

// Create a new cursor that will be used to read rows
func (t *partition_usageTable) CreateReader() rpc.ReaderInterface {
	return &partition_usageCursor{}
}

// Return a slice of rows that will be returned to Anyquery and filtered.
// The second return value is true if the cursor has no more rows to return
//
// The constraints are used for optimization purposes to "pre-filter" the rows
// If the rows returned don't match the constraints, it's not an issue. Anyquery will filter them out
func (t *partition_usageCursor) Query(constraints rpc.QueryConstraint) ([][]interface{}, bool, error) {
	// Example: Extract the name from the constraints
	// name := constraints.GetColumnConstraint(0).GetStringValue()
	// if name == "" {
	// 	return nil, true, fmt.Errorf("name must be set")
	// }

	mountpoint := constraints.GetColumnConstraint(0).GetStringValue()
	if mountpoint == "" {
		return nil, true, fmt.Errorf("mountpoint must be set. Example: SELECT * FROM system_partition_usage('/)")
	}

	usage, err := disk.Usage(mountpoint)
	if err != nil {
		return nil, true, fmt.Errorf("failed to get usage for %s: %w", mountpoint, err)
	}

	return [][]interface{}{
		{
			usage.Fstype,
			usage.Total,
			usage.Free,
			usage.Used,
			usage.UsedPercent,
			usage.InodesTotal,
			usage.InodesUsed,
			usage.InodesFree,
			usage.InodesUsedPercent,
		},
	}, true, nil
}

// A slice of rows to insert
func (t *partition_usageTable) Insert(rows [][]interface{}) error {
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
func (t *partition_usageTable) Update(rows [][]interface{}) error {
	return nil
}

// A slice of primary keys to delete
func (t *partition_usageTable) Delete(primaryKeys []interface{}) error {
	return nil
}

// A destructor to clean up resources
func (t *partition_usageTable) Close() error {
	return nil
}
