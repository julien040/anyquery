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
func process_statusCreator(args rpc.TableCreatorArgs) (rpc.Table, *rpc.DatabaseSchema, error) {
	// Example: get a token from the user configuration
	// token := args.UserConfig.GetString("token")
	// if token == "" {
	// 	return nil, nil, fmt.Errorf("token must be set in the plugin configuration")
	// }

	// Example: open a cache connection
	/* cache, err := helper.NewCache(helper.NewCacheArgs{
		Paths:         []string{"process_status", "process_status" + "_cache"},
		EncryptionKey: []byte("my_secret_key"),
	})*/

	return &process_statusTable{}, &rpc.DatabaseSchema{
		Columns: []rpc.DatabaseSchemaColumn{
			{
				Name:        "pid",
				Type:        rpc.ColumnTypeInt,
				IsParameter: true,
				IsRequired:  true,
			},
			{
				Name: "status",
				Type: rpc.ColumnTypeString,
			},
		},
	}, nil
}

// The table struct
// There is one per connection to the plugin and is created by the creator function
// In there, you can store any state you need to read the rows (e.g. a database connection, an API token, etc.)
type process_statusTable struct {
}

// The cursor struct
// There is one per query and is created by the createReader function
// In there, you can store any state you need to read the rows (e.g. a database connection from process_statusTable, an offset, a cursor, etc.)
type process_statusCursor struct {
}

// Create a new cursor that will be used to read rows
func (t *process_statusTable) CreateReader() rpc.ReaderInterface {
	return &process_statusCursor{}
}

// Return a slice of rows that will be returned to Anyquery and filtered.
// The second return value is true if the cursor has no more rows to return
//
// The constraints are used for optimization purposes to "pre-filter" the rows
// If the rows returned don't match the constraints, it's not an issue. Anyquery will filter them out
func (t *process_statusCursor) Query(constraints rpc.QueryConstraint) ([][]interface{}, bool, error) {
	pid := constraints.GetColumnConstraint(0).GetIntValue()
	if pid == 0 {
		return nil, true, nil
	}

	proc, err := process.NewProcess(int32(pid))
	if err != nil {
		return nil, true, fmt.Errorf("process not found")
	}

	statusArr, err := proc.Status()
	if err != nil {
		return nil, true, fmt.Errorf("status not found")
	}

	if len(statusArr) == 0 {
		return nil, true, fmt.Errorf("status not found")
	}

	return [][]interface{}{
		{statusArr[0]},
	}, true, nil

}

// A slice of rows to insert
func (t *process_statusTable) Insert(rows [][]interface{}) error {
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
func (t *process_statusTable) Update(rows [][]interface{}) error {
	return nil
}

// A slice of primary keys to delete
func (t *process_statusTable) Delete(primaryKeys []interface{}) error {
	return nil
}

// A destructor to clean up resources
func (t *process_statusTable) Close() error {
	return nil
}
