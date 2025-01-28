package main

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/dgraph-io/badger/v4"
	"github.com/google/go-github/v63/github"
	"github.com/julien040/anyquery/rpc"
)

// A constructor to create a new table instance
// This function is called everytime a new connection is made to the plugin
//
// It should return a new table instance, the database schema and if there is an error
func stargazers_repoCreator(args rpc.TableCreatorArgs) (rpc.Table, *rpc.DatabaseSchema, error) {
	// Request a connection
	client, token, err := getClient(args)
	if err != nil {
		return nil, nil, err
	}

	// Open the database
	db, err := openDatabase("stars_from_repo", token)
	if err != nil {
		return nil, nil, err
	}
	return &stargazers_repoTable{
			client: client,
			db:     db,
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
					Name:        "login",
					Type:        rpc.ColumnTypeString,
					Description: "The username of the stargazer",
				},
				{
					Name:        "starred_at",
					Type:        rpc.ColumnTypeDateTime,
					Description: "The date when the user starred the repository",
				},
				{
					Name:        "user_id",
					Type:        rpc.ColumnTypeString,
					Description: "The ID of the user",
				},
			},
		}, nil
}

type stargazers_repoTable struct {
	client *github.Client
	db     *badger.DB
}

type stargazers_repoCursor struct {
	client *github.Client
	db     *badger.DB
	pageID int
}

// Return a slice of rows that will be returned to Anyquery and filtered.
// The second return value is true if the cursor has no more rows to return
//
// The constraints are used for optimization purposes to "pre-filter" the rows
// If the rows returned don't match the constraints, it's not an issue. Anyquery will filter them out
func (t *stargazers_repoCursor) Query(constraints rpc.QueryConstraint) ([][]interface{}, bool, error) {
	repository := retrieveArgString(constraints, 0)

	if repository == "" {
		return nil, true, fmt.Errorf("missing repository")
	}

	cacheKey := fmt.Sprintf("stars-repo-%d-%s", t.pageID, repository)

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

	// Retrieve the stargazers
	stargazers, _, err := t.client.Activity.ListStargazers(context.Background(), owner, name, &github.ListOptions{
		Page:    t.pageID,
		PerPage: 100,
	})
	if err != nil {
		return nil, true, fmt.Errorf("error while retrieving stargazers: %v", err)
	}

	for _, stargazer := range stargazers {
		if stargazer == nil || stargazer.GetUser() == nil {
			continue
		}
		rows = append(rows, []interface{}{
			stargazer.GetUser().GetLogin(),
			stargazer.GetStarredAt().Format(time.RFC3339),
			stargazer.GetUser().GetID(),
		})
	}

	// Save the repositories in the cache
	err = saveCache(t.db, cacheKey, rows)
	if err != nil {
		log.Printf("failed to save cache(key=%s): %v", cacheKey, err)
	}

	t.pageID++

	return rows, len(rows) < 100, nil
}

// Create a new cursor that will be used to read rows
func (t *stargazers_repoTable) CreateReader() rpc.ReaderInterface {
	return &stargazers_repoCursor{
		client: t.client,
		db:     t.db,
		pageID: 1,
	}
}

// A destructor to clean up resources
func (t *stargazers_repoTable) Close() error {
	return nil
}
