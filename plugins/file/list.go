package main

import (
	"container/list"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/julien040/anyquery/rpc"
)

// A constructor to create a new table instance
// This function is called everytime a new connection is made to the plugin
//
// It should return a new table instance, the database schema and if there is an error
func listCreator(args rpc.TableCreatorArgs) (rpc.Table, *rpc.DatabaseSchema, error) {
	return &listTable{}, &rpc.DatabaseSchema{
		Columns: []rpc.DatabaseSchemaColumn{
			{
				Name:        "directory",
				Type:        rpc.ColumnTypeString,
				IsParameter: true,
				IsRequired:  true,
				Description: "The directory to list files from",
			},
			{
				Name:        "path",
				Type:        rpc.ColumnTypeString,
				Description: "The full path of the file relative to the directory",
			},
			{
				Name:        "file_name",
				Type:        rpc.ColumnTypeString,
				Description: "The name of the file",
			},
			{
				Name:        "file_type",
				Type:        rpc.ColumnTypeString,
				Description: "The extension of the file (after the last dot)",
			},
			{
				Name:        "size",
				Type:        rpc.ColumnTypeInt,
				Description: "The size of the file in bytes",
			},
			{
				Name:        "last_modified",
				Type:        rpc.ColumnTypeDateTime,
				Description: "The last modified time of the file (RFC3339)",
			},
			{
				Name:        "is_directory",
				Type:        rpc.ColumnTypeBool,
				Description: "If the file is a directory",
			},
		},
	}, nil
}

type listTable struct {
}

type listCursor struct {
	dirQueue list.List
	inited   bool
}

// Return a slice of rows that will be returned to Anyquery and filtered.
// The second return value is true if the cursor has no more rows to return
//
// The constraints are used for optimization purposes to "pre-filter" the rows
// If the rows returned don't match the constraints, it's not an issue. Anyquery will filter them out
func (t *listCursor) Query(constraints rpc.QueryConstraint) ([][]interface{}, bool, error) {
	var err error
	directory := ""
	for _, c := range constraints.Columns {
		if c.ColumnID == 0 {
			switch c.Value.(type) {
			case string:
				directory = c.Value.(string)
			default:
				return nil, true, fmt.Errorf("directory must be a string")
			}
		}
	}
	if directory == "" {
		return nil, true, fmt.Errorf("directory is required")
	}

	// Read the directory
	if !t.inited {
		t.inited = true
		// Ensure the directory exists
		var file os.FileInfo
		if file, err = os.Stat(directory); os.IsNotExist(err) {
			return nil, true, fmt.Errorf("directory does not exist")
		}
		if !file.IsDir() {
			return nil, true, fmt.Errorf("directory is not a directory")
		}
		t.dirQueue.PushBack(directory)
	}

	rows := make([][]interface{}, 0)

	if t.dirQueue.Len() == 0 {
		return rows, true, nil
	}

	// Read the directory
	dir := t.dirQueue.Front().Value.(string)
	t.dirQueue.Remove(t.dirQueue.Front())

	var files []os.DirEntry
	if files, err = os.ReadDir(dir); err != nil {
		return nil, true, fmt.Errorf("failed to read directory %s: %s", dir, err)
	}

	for _, file := range files {
		if file.IsDir() {
			t.dirQueue.PushBack(dir + "/" + file.Name())
			continue
		}

		splitted := strings.Split(file.Name(), ".")
		fileType := ""
		if len(splitted) > 1 {
			fileType = splitted[len(splitted)-1]
		}

		fileInfo, err := file.Info()
		if err != nil {
			continue
		}

		rows = append(rows, []interface{}{
			directory + "/" + file.Name(),
			file.Name(),
			fileType,
			fileInfo.Size(),
			fileInfo.ModTime().Format(time.RFC3339),
			file.IsDir(),
		})
	}

	return rows, t.dirQueue.Len() == 0, nil
}

// Create a new cursor that will be used to read rows
func (t *listTable) CreateReader() rpc.ReaderInterface {
	return &listCursor{}
}

// A destructor to clean up resources
func (t *listTable) Close() error {
	return nil
}
