package main

import (
	"context"
	"fmt"
	"log"

	"github.com/dgraph-io/badger/v4"
	"github.com/google/go-github/v63/github"
	"github.com/julien040/anyquery/rpc"
)

// A constructor to create a new table instance
// This function is called everytime a new connection is made to the plugin
//
// It should return a new table instance, the database schema and if there is an error
func repoListByUserCreator(args rpc.TableCreatorArgs) (rpc.Table, *rpc.DatabaseSchema, error) {
	// Request a connection
	client, token, err := getClient(args)
	if err != nil {
		return nil, nil, err
	}

	columns := []rpc.DatabaseSchemaColumn{
		{
			Name:        "user",
			Type:        rpc.ColumnTypeString,
			IsParameter: true,
			IsRequired:  true,
			Description: "The user to get the repositories from",
		},
	}
	columns = append(columns, repositorySchema...)

	// Open the database
	db, err := openDatabase("repo_by_user", token)
	if err != nil {
		return nil, nil, err
	}
	return &repoListByUserTable{
			client: client,
			db:     db,
		}, &rpc.DatabaseSchema{
			PrimaryKey: 1,
			Columns:    columns,
		}, nil
}

type repoListByUserTable struct {
	client *github.Client
	db     *badger.DB
}

type repoListByUserCursor struct {
	client *github.Client
	pageID int
	db     *badger.DB
}

// Return a slice of rows that will be returned to Anyquery and filtered.
// The second return value is true if the cursor has no more rows to return
//
// The constraints are used for optimization purposes to "pre-filter" the rows
// If the rows returned don't match the constraints, it's not an issue. Anyquery will filter them out
func (t *repoListByUserCursor) Query(constraints rpc.QueryConstraint) ([][]interface{}, bool, error) {
	user := retrieveArgString(constraints, 0)

	if user == "" {
		return nil, true, fmt.Errorf("missing user")
	}

	cacheKey := fmt.Sprintf("repositories-%d-%s", t.pageID, user)

	// Retrieve the repositories from the cache
	rows := [][]interface{}{}
	err := loadCache(t.db, cacheKey, &rows)
	if err == nil {
		t.pageID++
		return rows, len(rows) == 0, nil
	}

	// Retrieve the repositories from the API
	repos, _, err := t.client.Repositories.ListByUser(context.Background(), user, &github.RepositoryListByUserOptions{
		ListOptions: github.ListOptions{
			Page:    t.pageID,
			PerPage: 100,
		},
	})

	if err != nil {
		return nil, true, fmt.Errorf("error with github api: %w", err)
	}

	// Serialize the custom properties
	rows = convertRepo(repos, rows)

	// Save the cache
	err = saveCache(t.db, cacheKey, rows)
	if err != nil {
		log.Printf("Failed to save cache: %v", err)
	}

	t.pageID++

	return rows, len(rows) == 0, nil
}

// Create a new cursor that will be used to read rows
func (t *repoListByUserTable) CreateReader() rpc.ReaderInterface {
	return &repoListByUserCursor{
		client: t.client,
		pageID: 1,
		db:     t.db,
	}
}

// A destructor to clean up resources
func (t *repoListByUserTable) Close() error {
	return nil
}
