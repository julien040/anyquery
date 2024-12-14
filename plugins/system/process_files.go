package main

import (
	"fmt"
	"runtime"

	"github.com/julien040/anyquery/rpc"
	"github.com/shirou/gopsutil/v4/process"
)

// A constructor to create a new table instance
// This function is called everytime a new connection is made to the plugin
//
// It should return a new table instance, the database schema and if there is an error
func process_filesCreator(args rpc.TableCreatorArgs) (rpc.Table, *rpc.DatabaseSchema, error) {
	return &process_filesTable{}, &rpc.DatabaseSchema{
		Columns: []rpc.DatabaseSchemaColumn{
			{
				Name:        "pid",
				Type:        rpc.ColumnTypeInt,
				IsParameter: true,
				IsRequired:  true,
			},
			{
				Name: "path",
				Type: rpc.ColumnTypeString,
			},
			{
				Name: "file_descriptor",
				Type: rpc.ColumnTypeInt,
			},
		},
	}, nil
}

// The table struct
// There is one per connection to the plugin and is created by the creator function
// In there, you can store any state you need to read the rows (e.g. a database connection, an API token, etc.)
type process_filesTable struct {
}

// The cursor struct
// There is one per query and is created by the createReader function
// In there, you can store any state you need to read the rows (e.g. a database connection from process_filesTable, an offset, a cursor, etc.)
type process_filesCursor struct {
}

// Create a new cursor that will be used to read rows
func (t *process_filesTable) CreateReader() rpc.ReaderInterface {
	return &process_filesCursor{}
}

// Return a slice of rows that will be returned to Anyquery and filtered.
// The second return value is true if the cursor has no more rows to return
//
// The constraints are used for optimization purposes to "pre-filter" the rows
// If the rows returned don't match the constraints, it's not an issue. Anyquery will filter them out
func (t *process_filesCursor) Query(constraints rpc.QueryConstraint) ([][]interface{}, bool, error) {
	// Return error on darwin
	if runtime.GOOS != "linux" && runtime.GOOS != "windows" {
		return nil, true, fmt.Errorf("not supported on %s", runtime.GOOS)
	}
	pid := constraints.GetColumnConstraint(0).GetIntValue()
	if pid == 0 {
		return nil, true, nil
	}

	proc, err := process.NewProcess(int32(pid))
	if err != nil {
		return nil, true, fmt.Errorf("process not found")
	}

	files, err := proc.OpenFiles()
	if err != nil {
		return nil, true, fmt.Errorf("failed to get open files: %w", err)
	}

	rows := make([][]interface{}, 0, len(files))
	for _, file := range files {
		rows = append(rows, []interface{}{
			file.Path,
			file.Fd,
		})
	}

	return rows, true, nil
}

// A slice of rows to insert
func (t *process_filesTable) Insert(rows [][]interface{}) error {
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
func (t *process_filesTable) Update(rows [][]interface{}) error {
	return nil
}

// A slice of primary keys to delete
func (t *process_filesTable) Delete(primaryKeys []interface{}) error {
	return nil
}

// A destructor to clean up resources
func (t *process_filesTable) Close() error {
	return nil
}
