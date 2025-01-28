package main

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/dgraph-io/badger/v4"
	"github.com/google/go-github/v63/github"
	"github.com/julien040/anyquery/rpc"
)

// A constructor to create a new table instance
// This function is called everytime a new connection is made to the plugin
//
// It should return a new table instance, the database schema and if there is an error
func branchesCreator(args rpc.TableCreatorArgs) (rpc.Table, *rpc.DatabaseSchema, error) {
	// Request a connection
	client, token, err := getClient(args)
	if err != nil {
		return nil, nil, err
	}

	// Open the database
	db, err := openDatabase("branches", token)
	if err != nil {
		return nil, nil, err
	}
	return &branchesTable{
			client, db,
		}, &rpc.DatabaseSchema{
			PrimaryKey: 1,
			Columns: []rpc.DatabaseSchemaColumn{
				{
					Name:        "repository",
					Type:        rpc.ColumnTypeString,
					IsParameter: true,
					IsRequired:  true,
					Description: "The repository in the format owner/name",
				},
				{
					Name:        "name",
					Type:        rpc.ColumnTypeString,
					Description: "The name of the branch",
				},
				{
					Name:        "commit_sha",
					Type:        rpc.ColumnTypeString,
					Description: "The SHA of the commit the branch is pointing to",
				},

				{
					Name:        "protected",
					Type:        rpc.ColumnTypeBool,
					Description: "Whether the branch is protected (changes are restricted)",
				},
				{
					Name:        "url",
					Type:        rpc.ColumnTypeString,
					Description: "The URL to see the branch on GitHub",
				},
			},
		}, nil
}

type branchesTable struct {
	client *github.Client
	db     *badger.DB
}

type branchesCursor struct {
	client *github.Client
	db     *badger.DB
	pageID int
}

// Return a slice of rows that will be returned to Anyquery and filtered.
// The second return value is true if the cursor has no more rows to return
//
// The constraints are used for optimization purposes to "pre-filter" the rows
// If the rows returned don't match the constraints, it's not an issue. Anyquery will filter them out
func (t *branchesCursor) Query(constraints rpc.QueryConstraint) ([][]interface{}, bool, error) {
	repository := retrieveArgString(constraints, 0)

	if repository == "" {
		return nil, true, fmt.Errorf("missing repository")
	}

	cacheKey := fmt.Sprintf("branches-%d-%s", t.pageID, repository)

	// Retrieve the repositories from the cache
	rows := [][]interface{}{}
	err := loadCache(t.db, cacheKey, &rows)
	if err == nil {
		t.pageID++
		return rows, len(rows) == 0, nil
	}

	splitted := strings.Split(repository, "/")
	if len(splitted) != 2 {
		return nil, true, fmt.Errorf("repository must be in the format owner/name")
	}

	owner := splitted[0]
	name := splitted[1]

	// Retrieve the branches
	branches, _, err := t.client.Repositories.ListBranches(context.Background(), owner, name, &github.BranchListOptions{
		ListOptions: github.ListOptions{
			Page:    t.pageID,
			PerPage: 100,
		},
	})
	if err != nil {
		return nil, true, fmt.Errorf("failed to retrieve branches: %v", err)
	}

	for _, branch := range branches {
		rows = append(rows, []interface{}{
			branch.GetName(),
			branch.GetCommit().GetSHA(),
			branch.GetProtected(),
			branch.GetCommit().GetURL(),
		})
	}

	err = saveCache(t.db, cacheKey, rows)
	if err != nil {
		log.Printf("failed to save cache: %v", err)
	}

	t.pageID++

	return rows, len(rows) == 0, nil
}

// Create a new cursor that will be used to read rows
func (t *branchesTable) CreateReader() rpc.ReaderInterface {
	return &branchesCursor{
		client: t.client,
		db:     t.db,
		pageID: 1,
	}
}

// A destructor to clean up resources
func (t *branchesTable) Close() error {
	return nil
}
