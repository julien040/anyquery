package main

import (
	"fmt"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/julien040/anyquery/rpc"
	"github.com/julien040/anyquery/rpc/helper"
)

// A constructor to create a new table instance
// This function is called everytime a new connection is made to the plugin
//
// It should return a new table instance, the database schema and if there is an error
func commitsCreator(args rpc.TableCreatorArgs) (rpc.Table, *rpc.DatabaseSchema, error) {
	return &commitsTable{}, &rpc.DatabaseSchema{
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
				Name:        "hash",
				Type:        rpc.ColumnTypeString,
				Description: "The hash of the commit",
			},
			{
				Name:        "author_name",
				Type:        rpc.ColumnTypeString,
				Description: "The name of the author of the commit",
			},
			{
				Name:        "author_email",
				Type:        rpc.ColumnTypeString,
				Description: "The email of the author of the commit",
			},
			{
				Name:        "author_date",
				Type:        rpc.ColumnTypeString,
				Description: "The date when the commit was authored",
			},
			{
				Name:        "committer_name",
				Type:        rpc.ColumnTypeString,
				Description: "The name of the committer of the commit",
			},
			{
				Name:        "committer_email",
				Type:        rpc.ColumnTypeString,
				Description: "The email of the committer of the commit",
			},
			{
				Name:        "committer_date",
				Type:        rpc.ColumnTypeString,
				Description: "The date when the commit was committed",
			},
			{
				Name:        "message",
				Type:        rpc.ColumnTypeString,
				Description: "The content of the message commit (title + body)",
			},
			{
				Name:        "parents",
				Type:        rpc.ColumnTypeString,
				Description: "A JSON array of the hashes of the parent commits",
			},
		},
	}, nil
}

type commitsTable struct {
}

type commitsCursor struct {
	iter           object.CommitIter
	iterExhausted  bool
	repository     *git.Repository
	alreadyVisited map[string]bool
}

// Return a slice of rows that will be returned to Anyquery and filtered.
// The second return value is true if the cursor has no more rows to return
//
// The constraints are used for optimization purposes to "pre-filter" the rows
// If the rows returned don't match the constraints, it's not an issue. Anyquery will filter them out
func (t *commitsCursor) Query(constraints rpc.QueryConstraint) ([][]interface{}, bool, error) {
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

	// Create the iterator if it's not already created
	if t.iter == nil && !t.iterExhausted {
		iter, err := t.repository.CommitObjects()
		if err != nil {
			return nil, true, fmt.Errorf("error getting commits: %s", err)
		}
		t.iter = iter
	}

	// Get the next 128 commits and return them
	rows := make([][]interface{}, 0, 128)
	for i := 0; i < 128; i++ {
		commit, err := t.iter.Next()
		if err != nil {
			if err.Error() == "EOF" {
				t.iterExhausted = true
				break
			}
			return nil, true, fmt.Errorf("error getting next commit: %s", err)
		}
		if commit == nil {
			t.iterExhausted = true
			break
		}

		if t.alreadyVisited[commit.Hash.String()] {
			continue
		}
		t.alreadyVisited[commit.Hash.String()] = true

		var parents []string
		for _, parent := range commit.ParentHashes {
			parents = append(parents, parent.String())
		}

		rows = append(rows, []interface{}{
			commit.Hash.String(),
			commit.Author.Name,
			commit.Author.Email,
			commit.Author.When.Format(time.RFC3339),
			commit.Committer.Name,
			commit.Committer.Email,
			commit.Committer.When.Format(time.RFC3339),
			commit.Message,
			helper.Serialize(parents),
		})

	}

	// Do a bit of cleanup because this cursor is not going to be used anymore
	// but a reference to the cursor will still be kept by Anyquery
	// So we do a bit of cleanup for the garbage collector
	if t.iterExhausted {
		t.alreadyVisited = nil
		t.repository = nil
		t.iter.Close()
	}

	return rows, t.iterExhausted, nil
}

// Create a new cursor that will be used to read rows
func (t *commitsTable) CreateReader() rpc.ReaderInterface {
	return &commitsCursor{
		alreadyVisited: make(map[string]bool),
	}
}

// A destructor to clean up resources
func (t *commitsTable) Close() error {
	return nil
}
