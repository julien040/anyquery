package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/dgraph-io/badger/v4"
	"github.com/google/go-github/v63/github"
	"github.com/julien040/anyquery/rpc"
)

// A constructor to create a new table instance
// This function is called everytime a new connection is made to the plugin
//
// It should return a new table instance, the database schema and if there is an error
func tagsCreator(args rpc.TableCreatorArgs) (rpc.Table, *rpc.DatabaseSchema, error) {
	// Request a connection
	client, token, err := getClient(args)
	if err != nil {
		return nil, nil, err
	}

	// Open the database
	db, err := openDatabase("tags", token)
	if err != nil {
		return nil, nil, err
	}
	return &tagsTable{
			client, db,
		}, &rpc.DatabaseSchema{
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
					Name: "commit_sha",
					Type: rpc.ColumnTypeString,
				},
				{
					Name: "commit_url",
					Type: rpc.ColumnTypeString,
				},
			},
		}, nil
}

type tagsTable struct {
	client *github.Client
	db     *badger.DB
}

type tagsCursor struct {
	client *github.Client
	db     *badger.DB
	pageID int
}

// Return a slice of rows that will be returned to Anyquery and filtered.
// The second return value is true if the cursor has no more rows to return
//
// The constraints are used for optimization purposes to "pre-filter" the rows
// If the rows returned don't match the constraints, it's not an issue. Anyquery will filter them out
func (t *tagsCursor) Query(constraints rpc.QueryConstraint) ([][]interface{}, bool, error) {
	repository := retrieveArgString(constraints, 0)

	if repository == "" {
		return nil, true, fmt.Errorf("missing repository")
	}

	cacheKey := fmt.Sprintf("tags-%d-%s", t.pageID, repository)

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

	tags, _, err := t.client.Repositories.ListTags(context.Background(), owner, name, &github.ListOptions{
		Page:    t.pageID,
		PerPage: 100,
	})

	if err != nil {
		return nil, true, fmt.Errorf("failed to retrieve tags: %w", err)
	}

	for _, tag := range tags {
		rows = append(rows, []interface{}{
			tag.GetName(),
			tag.GetCommit().GetSHA(),
			fmt.Sprintf("https://github.com/%s/%s/commit/%s", owner, name, tag.GetCommit().GetSHA()),
		})
	}

	err = saveCache(t.db, cacheKey, rows)
	if err != nil {
		return nil, true, fmt.Errorf("failed to save cache: %w", err)
	}

	t.pageID++

	return rows, len(rows) == 0, nil
}

// Create a new cursor that will be used to read rows
func (t *tagsTable) CreateReader() rpc.ReaderInterface {
	return &tagsCursor{
		client: t.client,
		db:     t.db,
	}
}

// A slice of rows to insert
func (t *tagsTable) Insert(rows [][]interface{}) error {
	return nil
}

// A slice of rows to update
// The first element of each row is the primary key
// while the rest are the values to update
// The primary key is therefore present twice
func (t *tagsTable) Update(rows [][]interface{}) error {
	return nil
}

// A slice of primary keys to delete
func (t *tagsTable) Delete(primaryKeys []interface{}) error {
	return nil
}

// A destructor to clean up resources
func (t *tagsTable) Close() error {
	return nil
}
