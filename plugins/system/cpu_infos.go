package main

import (
	"fmt"

	"github.com/julien040/anyquery/rpc"
	"github.com/julien040/anyquery/rpc/helper"
	"github.com/shirou/gopsutil/v4/cpu"
)

// A constructor to create a new table instance
// This function is called everytime a new connection is made to the plugin
//
// It should return a new table instance, the database schema and if there is an error
func cpu_infosCreator(args rpc.TableCreatorArgs) (rpc.Table, *rpc.DatabaseSchema, error) {
	return &cpu_infosTable{}, &rpc.DatabaseSchema{
		Columns: []rpc.DatabaseSchemaColumn{
			{
				Name: "cpu_name",
				Type: rpc.ColumnTypeString,
			},
			{
				Name: "cpu_vendor_id",
				Type: rpc.ColumnTypeString,
			},
			{
				Name: "cpu_family",
				Type: rpc.ColumnTypeString,
			},
			{
				Name: "cpu_model",
				Type: rpc.ColumnTypeString,
			},
			{
				Name: "cpu_stepping",
				Type: rpc.ColumnTypeInt,
			},
			{
				Name: "cpu_physical_id",
				Type: rpc.ColumnTypeString,
			},
			{
				Name: "cpu_core_id",
				Type: rpc.ColumnTypeString,
			},
			{
				Name: "cpu_cores",
				Type: rpc.ColumnTypeInt,
			},
			{
				Name: "cpu_model_name",
				Type: rpc.ColumnTypeString,
			},
			{
				Name: "cpu_frequency",
				Type: rpc.ColumnTypeInt,
			},
			{
				Name: "cpu_cache_size",
				Type: rpc.ColumnTypeInt,
			},
			{
				Name: "cpu_flags",
				Type: rpc.ColumnTypeString,
			},
			{
				Name: "cpu_microcode",
				Type: rpc.ColumnTypeString,
			},
		},
	}, nil
}

// The table struct
// There is one per connection to the plugin and is created by the creator function
// In there, you can store any state you need to read the rows (e.g. a database connection, an API token, etc.)
type cpu_infosTable struct {
}

// The cursor struct
// There is one per query and is created by the createReader function
// In there, you can store any state you need to read the rows (e.g. a database connection from cpu_infosTable, an offset, a cursor, etc.)
type cpu_infosCursor struct {
}

// Create a new cursor that will be used to read rows
func (t *cpu_infosTable) CreateReader() rpc.ReaderInterface {
	return &cpu_infosCursor{}
}

// Return a slice of rows that will be returned to Anyquery and filtered.
// The second return value is true if the cursor has no more rows to return
//
// The constraints are used for optimization purposes to "pre-filter" the rows
// If the rows returned don't match the constraints, it's not an issue. Anyquery will filter them out
func (t *cpu_infosCursor) Query(constraints rpc.QueryConstraint) ([][]interface{}, bool, error) {
	infoStats, err := cpu.Info()
	if err != nil {
		return nil, true, fmt.Errorf("cpu info not found: %w", err)
	}

	rows := make([][]interface{}, 0, len(infoStats))
	for _, info := range infoStats {
		rows = append(rows, []interface{}{
			info.CPU,
			info.VendorID,
			info.Family,
			info.Model,
			info.Stepping,
			info.PhysicalID,
			info.CoreID,
			info.Cores,
			info.ModelName,
			info.Mhz,
			info.CacheSize,
			helper.Serialize(info.Flags),
			info.Microcode,
		})
	}

	return rows, true, nil
}

// A slice of rows to insert
func (t *cpu_infosTable) Insert(rows [][]interface{}) error {
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
func (t *cpu_infosTable) Update(rows [][]interface{}) error {
	return nil
}

// A slice of primary keys to delete
func (t *cpu_infosTable) Delete(primaryKeys []interface{}) error {
	return nil
}

// A destructor to clean up resources
func (t *cpu_infosTable) Close() error {
	return nil
}
