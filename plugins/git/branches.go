package main

import (
	"fmt"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/julien040/anyquery/rpc"
)

// A constructor to create a new table instance
// This function is called everytime a new connection is made to the plugin
//
// It should return a new table instance, the database schema and if there is an error
func branchesCreator(args rpc.TableCreatorArgs) (rpc.Table, *rpc.DatabaseSchema, error) {
	return &branchesTable{}, &rpc.DatabaseSchema{
		PrimaryKey: 1,
		Columns: []rpc.DatabaseSchemaColumn{
			{
				Name:        "repository",
				Type:        rpc.ColumnTypeString,
				IsParameter: true,
				IsRequired:  true,
				Description: "The path to the repository. Can be a local path (e.g. /path/to/repo) or a URL (e.g. https://github.com/julien040/anyquery.git)",
			},
			{
				Name:        "full_name",
				Type:        rpc.ColumnTypeString,
				Description: "The full name of the branch. For example, refs/heads/master",
			},
			{
				Name:        "name",
				Type:        rpc.ColumnTypeString,
				Description: "The short name of the branch. For example, master",
			},
			{
				Name:        "hash",
				Type:        rpc.ColumnTypeString,
				Description: "The hash of the commit the branch is pointing to",
			},
		},
	}, nil
}

type branchesTable struct {
}

type branchesCursor struct {
	repository *git.Repository
}

// Return a slice of rows that will be returned to Anyquery and filtered.
// The second return value is true if the cursor has no more rows to return
//
// The constraints are used for optimization purposes to "pre-filter" the rows
// If the rows returned don't match the constraints, it's not an issue. Anyquery will filter them out
func (t *branchesCursor) Query(constraints rpc.QueryConstraint) ([][]interface{}, bool, error) {
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
	branches, err := t.repository.Branches()
	if err != nil {
		return nil, true, fmt.Errorf("error getting branches: %s", err)
	}

	branches.ForEach(func(r *plumbing.Reference) error {
		rows = append(rows, []interface{}{
			r.Name().String(),
			r.Name().Short(),
			r.Hash().String(),
		})
		return nil
	})

	return rows, true, nil
}

// Create a new cursor that will be used to read rows
func (t *branchesTable) CreateReader() rpc.ReaderInterface {
	return &branchesCursor{}
}

// A destructor to clean up resources
func (t *branchesTable) Close() error {
	return nil
}
