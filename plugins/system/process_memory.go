package main

import (
	"fmt"

	"github.com/julien040/anyquery/rpc"
	"github.com/shirou/gopsutil/v4/process"
)

// A constructor to create a new table instance
// This function is called everytime a new connection is made to the plugin
//
// It should return a new table instance, the database schema and if there is an error
func process_memoryCreator(args rpc.TableCreatorArgs) (rpc.Table, *rpc.DatabaseSchema, error) {
	// Example: get a token from the user configuration
	// token := args.UserConfig.GetString("token")
	// if token == "" {
	// 	return nil, nil, fmt.Errorf("token must be set in the plugin configuration")
	// }

	// Example: open a cache connection
	/* cache, err := helper.NewCache(helper.NewCacheArgs{
		Paths:         []string{"process_memory", "process_memory" + "_cache"},
		EncryptionKey: []byte("my_secret_key"),
	})*/

	return &process_memoryTable{}, &rpc.DatabaseSchema{
		Columns: []rpc.DatabaseSchemaColumn{
			{
				Name:        "pid",
				Type:        rpc.ColumnTypeInt,
				IsParameter: true,
				IsRequired:  true,
			},
			{
				Name: "resident_set_size",
				Type: rpc.ColumnTypeInt,
			},
			{
				Name: "virtual_memory_size",
				Type: rpc.ColumnTypeInt,
			},
			{ // HWM
				Name: "high_water_mark",
				Type: rpc.ColumnTypeInt,
			},
			{
				Name: "data",
				Type: rpc.ColumnTypeInt,
			},
			{
				Name: "stack",
				Type: rpc.ColumnTypeInt,
			},
			{
				Name: "locked",
				Type: rpc.ColumnTypeInt,
			},
			{
				Name: "swap",
				Type: rpc.ColumnTypeInt,
			},
			{
				Name: "memory_percent",
				Type: rpc.ColumnTypeFloat,
			},
		},
	}, nil
}

// The table struct
// There is one per connection to the plugin and is created by the creator function
// In there, you can store any state you need to read the rows (e.g. a database connection, an API token, etc.)
type process_memoryTable struct {
}

// The cursor struct
// There is one per query and is created by the createReader function
// In there, you can store any state you need to read the rows (e.g. a database connection from process_memoryTable, an offset, a cursor, etc.)
type process_memoryCursor struct {
}

// Create a new cursor that will be used to read rows
func (t *process_memoryTable) CreateReader() rpc.ReaderInterface {
	return &process_memoryCursor{}
}

// Return a slice of rows that will be returned to Anyquery and filtered.
// The second return value is true if the cursor has no more rows to return
//
// The constraints are used for optimization purposes to "pre-filter" the rows
// If the rows returned don't match the constraints, it's not an issue. Anyquery will filter them out
func (t *process_memoryCursor) Query(constraints rpc.QueryConstraint) ([][]interface{}, bool, error) {
	pid := constraints.GetColumnConstraint(0).GetIntValue()
	if pid == 0 {
		return nil, true, nil
	}

	proc, err := process.NewProcess(int32(pid))
	if err != nil {
		return nil, true, fmt.Errorf("process not found")
	}

	memoryInf, err := proc.MemoryInfo()
	if err != nil {
		return nil, true, fmt.Errorf("memory info not found")
	}

	percentage, err := proc.MemoryPercent()
	if err != nil {
		return nil, true, fmt.Errorf("memory percent not found")
	}

	return [][]interface{}{
		{memoryInf.RSS, memoryInf.VMS, memoryInf.HWM, memoryInf.Data, memoryInf.Stack, memoryInf.Locked, memoryInf.Swap, percentage},
	}, true, nil
}

// A slice of rows to insert
func (t *process_memoryTable) Insert(rows [][]interface{}) error {
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
func (t *process_memoryTable) Update(rows [][]interface{}) error {
	return nil
}

// A slice of primary keys to delete
func (t *process_memoryTable) Delete(primaryKeys []interface{}) error {
	return nil
}

// A destructor to clean up resources
func (t *process_memoryTable) Close() error {
	return nil
}
