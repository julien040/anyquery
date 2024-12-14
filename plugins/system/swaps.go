package main

import (
	"fmt"

	"github.com/julien040/anyquery/rpc"
	"github.com/shirou/gopsutil/v4/mem"
)

// A constructor to create a new table instance
// This function is called everytime a new connection is made to the plugin
//
// It should return a new table instance, the database schema and if there is an error
func swapsCreator(args rpc.TableCreatorArgs) (rpc.Table, *rpc.DatabaseSchema, error) {
	return &swapsTable{}, &rpc.DatabaseSchema{
		Columns: []rpc.DatabaseSchemaColumn{
			{
				Name: "swap_name",
				Type: rpc.ColumnTypeString,
			},
			{
				Name: "total",
				Type: rpc.ColumnTypeInt,
			},
			{
				Name: "used",
				Type: rpc.ColumnTypeInt,
			},
			{
				Name: "free",
				Type: rpc.ColumnTypeInt,
			},
		},
	}, nil
}

// The table struct
// There is one per connection to the plugin and is created by the creator function
// In there, you can store any state you need to read the rows (e.g. a database connection, an API token, etc.)
type swapsTable struct {
}

// The cursor struct
// There is one per query and is created by the createReader function
// In there, you can store any state you need to read the rows (e.g. a database connection from swapsTable, an offset, a cursor, etc.)
type swapsCursor struct {
}

// Create a new cursor that will be used to read rows
func (t *swapsTable) CreateReader() rpc.ReaderInterface {
	return &swapsCursor{}
}

// Return a slice of rows that will be returned to Anyquery and filtered.
// The second return value is true if the cursor has no more rows to return
//
// The constraints are used for optimization purposes to "pre-filter" the rows
// If the rows returned don't match the constraints, it's not an issue. Anyquery will filter them out
func (t *swapsCursor) Query(constraints rpc.QueryConstraint) ([][]interface{}, bool, error) {
	swapsDevice, err := mem.SwapDevices()
	if err != nil {
		return nil, true, fmt.Errorf("failed to get swap devices: %w", err)
	}

	rows := make([][]interface{}, 0, len(swapsDevice))
	for _, swap := range swapsDevice {
		if swap == nil {
			continue
		}
		rows = append(rows, []interface{}{
			swap.Name,
			swap.FreeBytes + swap.UsedBytes,
			swap.UsedBytes,
			swap.FreeBytes,
		})
	}

	return rows, true, nil
}

// A slice of rows to insert
func (t *swapsTable) Insert(rows [][]interface{}) error {
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
func (t *swapsTable) Update(rows [][]interface{}) error {
	return nil
}

// A slice of primary keys to delete
func (t *swapsTable) Delete(primaryKeys []interface{}) error {
	return nil
}

// A destructor to clean up resources
func (t *swapsTable) Close() error {
	return nil
}
