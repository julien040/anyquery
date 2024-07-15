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
func statusCreator(args rpc.TableCreatorArgs) (rpc.Table, *rpc.DatabaseSchema, error) {
	return &statusTable{}, &rpc.DatabaseSchema{
		PrimaryKey: 1,
		Columns: []rpc.DatabaseSchemaColumn{
			{
				Name:        "repository",
				Type:        rpc.ColumnTypeString,
				IsParameter: true,
				IsRequired:  true,
			},
			{
				Name: "file_name",
				Type: rpc.ColumnTypeString,
			},
			{
				Name: "staging_status",
				Type: rpc.ColumnTypeString,
			},
			{
				Name: "worktree_status",
				Type: rpc.ColumnTypeString,
			},
		},
	}, nil
}

type statusTable struct {
}

type statusCursor struct {
	repository *git.Repository
}

// Return a slice of rows that will be returned to Anyquery and filtered.
// The second return value is true if the cursor has no more rows to return
//
// The constraints are used for optimization purposes to "pre-filter" the rows
// If the rows returned don't match the constraints, it's not an issue. Anyquery will filter them out
func (t *statusCursor) Query(constraints rpc.QueryConstraint) ([][]interface{}, bool, error) {
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

	wt, err := t.repository.Worktree()
	if err != nil {
		return nil, true, fmt.Errorf("failed to get worktree: %w", err)
	}

	status, err := wt.Status()
	if err != nil {
		return nil, true, fmt.Errorf("failed to get status: %w", err)
	}

	for file, stat := range status {
		staging := ""
		switch stat.Staging {
		case git.Unmodified:
			staging = "unmodified"
		case git.Untracked:
			staging = "untracked"
		case git.Added:
			staging = "added"
		case git.Deleted:
			staging = "deleted"
		case git.Modified:
			staging = "modified"
		case git.Renamed:
			staging = "renamed"
		case git.Copied:
			staging = "copied"
		case git.UpdatedButUnmerged:
			staging = "updated_but_unmerged"
		}

		worktree := ""
		switch stat.Worktree {
		case git.Unmodified:
			worktree = "unmodified"
		case git.Untracked:
			worktree = "untracked"
		case git.Added:
			worktree = "added"
		case git.Deleted:
			worktree = "deleted"
		case git.Modified:
			worktree = "modified"
		case git.Renamed:
			worktree = "renamed"
		case git.Copied:
			worktree = "copied"
		case git.UpdatedButUnmerged:
			worktree = "updated_but_unmerged"
		}

		rows = append(rows, []interface{}{
			file,
			staging,
			worktree,
		})
	}

	return rows, true, nil

}

// Create a new cursor that will be used to read rows
func (t *statusTable) CreateReader() rpc.ReaderInterface {
	return &statusCursor{}
}

// A slice of rows to insert
func (t *statusTable) Insert(rows [][]interface{}) error {
	return nil
}

// A slice of rows to update
// The first element of each row is the primary key
// while the rest are the values to update
// The primary key is therefore present twice
func (t *statusTable) Update(rows [][]interface{}) error {
	return nil
}

// A slice of primary keys to delete
func (t *statusTable) Delete(primaryKeys []interface{}) error {
	return nil
}

// A destructor to clean up resources
func (t *statusTable) Close() error {
	return nil
}
