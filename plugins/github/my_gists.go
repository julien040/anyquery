package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/dgraph-io/badger/v4"
	"github.com/google/go-github/v63/github"
	"github.com/julien040/anyquery/rpc"
)

// A constructor to create a new table instance
// This function is called everytime a new connection is made to the plugin
//
// It should return a new table instance, the database schema and if there is an error
func my_gistsCreator(args rpc.TableCreatorArgs) (rpc.Table, *rpc.DatabaseSchema, error) {
	// Request a connection
	client, token, err := getClient(args)
	if err != nil {
		return nil, nil, err
	}

	// Open the database
	db, err := openDatabase("my_gists", token)
	if err != nil {
		return nil, nil, err
	}
	return &my_gistsTable{client, db}, &rpc.DatabaseSchema{
		PrimaryKey: 1,
		Columns:    gistSchema,
	}, nil
}

type my_gistsTable struct {
	client *github.Client
	db     *badger.DB
}

type my_gistsCursor struct {
	client *github.Client
	db     *badger.DB
	pageID int
}

// Return a slice of rows that will be returned to Anyquery and filtered.
// The second return value is true if the cursor has no more rows to return
//
// The constraints are used for optimization purposes to "pre-filter" the rows
// If the rows returned don't match the constraints, it's not an issue. Anyquery will filter them out
func (t *my_gistsCursor) Query(constraints rpc.QueryConstraint) ([][]interface{}, bool, error) {
	cacheKey := fmt.Sprintf("gists-%d", t.pageID)

	// Check the cache
	rows := [][]interface{}{}
	err := loadCache(t.db, cacheKey, &rows)
	if err == nil {
		t.pageID++
		return rows, len(rows) == 0, nil
	}

	// Get the gists from the API
	gists, _, err := t.client.Gists.List(context.Background(), "", &github.GistListOptions{
		ListOptions: github.ListOptions{
			Page:    t.pageID,
			PerPage: 100,
		},
	})

	if err != nil {
		return nil, true, fmt.Errorf("failed to get gists: %w", err)
	}

	// Convert the gists to rows
	for _, gist := range gists {
		rows = append(rows, []interface{}{
			gist.GetID(),
			gist.GetHTMLURL(),
			gist.GetOwner().GetLogin(),
			gist.GetOwner().GetHTMLURL(),
			gist.GetDescription(),
			gist.GetComments(),
			gist.GetPublic(),
			gist.GetCreatedAt().Format(time.RFC3339),
			gist.GetUpdatedAt().Format(time.RFC3339),
		})
	}

	// Save the rows in the cache
	err = saveCache(t.db, cacheKey, rows)
	if err != nil {
		log.Printf("failed to save cache: %v", err)
	}

	t.pageID++

	return rows, len(rows) == 0, nil
}

// Create a new cursor that will be used to read rows
func (t *my_gistsTable) CreateReader() rpc.ReaderInterface {
	return &my_gistsCursor{
		client: t.client,
		db:     t.db,
		pageID: 1,
	}
}

// A slice of rows to insert
func (t *my_gistsTable) Insert(rows [][]interface{}) error {
	return nil
}

// A slice of rows to update
// The first element of each row is the primary key
// while the rest are the values to update
// The primary key is therefore present twice
func (t *my_gistsTable) Update(rows [][]interface{}) error {
	return nil
}

// A slice of primary keys to delete
func (t *my_gistsTable) Delete(primaryKeys []interface{}) error {
	return nil
}

// A destructor to clean up resources
func (t *my_gistsTable) Close() error {
	return nil
}
