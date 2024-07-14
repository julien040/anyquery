package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/julien040/anyquery/rpc"
)

// A constructor to create a new table instance
// This function is called everytime a new connection is made to the plugin
//
// It should return a new table instance, the database schema and if there is an error
func searchCreator(args rpc.TableCreatorArgs) (rpc.Table, *rpc.DatabaseSchema, error) {
	return &searchTable{}, &rpc.DatabaseSchema{
		PrimaryKey: 1,
		Columns: []rpc.DatabaseSchemaColumn{
			{
				Name:        "pattern",
				Type:        rpc.ColumnTypeString,
				IsParameter: true,
				IsRequired:  true,
			},
			{
				Name: "path",
				Type: rpc.ColumnTypeString,
			},
			{
				Name: "file_name",
				Type: rpc.ColumnTypeString,
			},
			{
				Name: "file_type",
				Type: rpc.ColumnTypeString,
			},
			{
				Name: "size",
				Type: rpc.ColumnTypeInt,
			},
			{
				Name: "last_modified",
				Type: rpc.ColumnTypeInt,
			},
			{
				Name: "is_directory",
				Type: rpc.ColumnTypeInt,
			},
		},
	}, nil
}

type searchTable struct {
}

type searchCursor struct {
	currentIndex int
	matches      []string
}

// Return a slice of rows that will be returned to Anyquery and filtered.
// The second return value is true if the cursor has no more rows to return
//
// The constraints are used for optimization purposes to "pre-filter" the rows
// If the rows returned don't match the constraints, it's not an issue. Anyquery will filter them out
func (t *searchCursor) Query(constraints rpc.QueryConstraint) ([][]interface{}, bool, error) {
	// Fail safe
	if t.currentIndex >= len(t.matches) && t.matches != nil {
		return nil, true, nil
	}

	// If the matches are not set, get them
	if t.matches == nil {
		// Get the constraints
		pattern := ""
		for _, c := range constraints.Columns {
			if c.ColumnID == 0 {
				switch c.Value.(type) {
				case string:
					pattern = c.Value.(string)
				default:
					return nil, true, fmt.Errorf("pattern must be a string")
				}
			}
		}

		if pattern == "" {
			return nil, true, fmt.Errorf("pattern is required")
		}

		// Get the files
		matches, err := filepath.Glob(pattern)
		if err != nil {
			return nil, true, fmt.Errorf("pattern is invalid: %s", err)
		}

		t.matches = matches
	}

	// Return the rows and process 100 records at a time
	rows := make([][]interface{}, 0)
	for i := t.currentIndex; i < len(t.matches) && i < t.currentIndex+100; i++ {
		fileInfo, err := os.Stat(t.matches[i])

		splitted := strings.Split(fileInfo.Name(), ".")
		fileType := ""
		if len(splitted) > 1 {
			fileType = splitted[len(splitted)-1]
		}

		if err != nil {
			log.Printf("Error getting file info for %s: %s", t.matches[i], err)
			continue
		}

		rows = append(rows, []interface{}{
			t.matches[i],
			fileInfo.Name(),
			fileType,
			fileInfo.Size(),
			fileInfo.ModTime().Format(time.RFC3339),
			fileInfo.IsDir(),
		})
	}

	t.currentIndex += 100

	return rows, t.currentIndex >= len(t.matches), nil
}

// Create a new cursor that will be used to read rows
func (t *searchTable) CreateReader() rpc.ReaderInterface {
	return &searchCursor{}
}

// A slice of rows to insert
func (t *searchTable) Insert(rows [][]interface{}) error {
	return nil
}

// A slice of rows to update
// The first element of each row is the primary key
// while the rest are the values to update
// The primary key is therefore present twice
func (t *searchTable) Update(rows [][]interface{}) error {
	return nil
}

// A slice of primary keys to delete
func (t *searchTable) Delete(primaryKeys []interface{}) error {
	return nil
}

// A destructor to clean up resources
func (t *searchTable) Close() error {
	return nil
}
