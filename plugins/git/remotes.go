package main

import (
	"fmt"

	"github.com/go-git/go-git/v5"
	"github.com/julien040/anyquery/rpc"
)

// A constructor to create a new table instance
// This function is called everytime a new connection is made to the plugin
//
// It should return a new table instance, the database schema and if there is an error
func remotesCreator(args rpc.TableCreatorArgs) (rpc.Table, *rpc.DatabaseSchema, error) {
	return &remotesTable{}, &rpc.DatabaseSchema{
		PrimaryKey: 1,
		Columns: []rpc.DatabaseSchemaColumn{
			{
				Name:        "repository",
				Type:        rpc.ColumnTypeString,
				IsParameter: true,
				IsRequired:  true,
			},
			{
				Name: "name",
				Type: rpc.ColumnTypeString,
			},
			{
				Name: "url",
				Type: rpc.ColumnTypeString,
			},
			{
				Name: "is_mirror",
				Type: rpc.ColumnTypeInt,
			},
		},
	}, nil
}

type remotesTable struct {
}

type remotesCursor struct {
	repository *git.Repository
}

// Return a slice of rows that will be returned to Anyquery and filtered.
// The second return value is true if the cursor has no more rows to return
//
// The constraints are used for optimization purposes to "pre-filter" the rows
// If the rows returned don't match the constraints, it's not an issue. Anyquery will filter them out
func (t *remotesCursor) Query(constraints rpc.QueryConstraint) ([][]interface{}, bool, error) {
	// Open the repository if it's not already opened
	if t.repository == nil {
		repoPath := ""
		for _, c := range constraints.Columns {
			if c.ColumnID == 0 {
				if parsed, ok := c.Value.(string); ok {
					repoPath = parsed
				}
			}
		}
		if repoPath == "" {
			return nil, true, fmt.Errorf("a repository of type string is required")
		}

		repo, err := openRepository(repoPath)
		if err != nil {
			return nil, true, err
		}
		t.repository = repo
	}

	rows := make([][]interface{}, 0)
	remotes, err := t.repository.Remotes()
	if err != nil {
		return nil, true, fmt.Errorf("error getting remotes: %s", err)
	}

	for _, remote := range remotes {
		rows = append(rows, []interface{}{
			remote.Config().Name,
			remote.Config().URLs,
			remote.Config().Mirror,
		})
	}

	return rows, true, nil
}

// Create a new cursor that will be used to read rows
func (t *remotesTable) CreateReader() rpc.ReaderInterface {
	return &remotesCursor{}
}

// A slice of rows to insert
func (t *remotesTable) Insert(rows [][]interface{}) error {
	return nil
}

// A slice of rows to update
// The first element of each row is the primary key
// while the rest are the values to update
// The primary key is therefore present twice
func (t *remotesTable) Update(rows [][]interface{}) error {
	return nil
}

// A slice of primary keys to delete
func (t *remotesTable) Delete(primaryKeys []interface{}) error {
	return nil
}

// A destructor to clean up resources
func (t *remotesTable) Close() error {
	return nil
}