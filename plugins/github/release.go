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
func releaseCreator(args rpc.TableCreatorArgs) (rpc.Table, *rpc.DatabaseSchema, error) {
	// Request a connection
	client, token, err := getClient(args)
	if err != nil {
		return nil, nil, err
	}

	// Open the database
	db, err := openDatabase("releases", token)
	if err != nil {
		return nil, nil, err
	}
	return &releaseTable{
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
					Name: "id",
					Type: rpc.ColumnTypeInt,
				},
				{
					Name: "name",
					Type: rpc.ColumnTypeString,
				},
				{
					Name: "tag",
					Type: rpc.ColumnTypeString,
				},
				{
					Name: "body",
					Type: rpc.ColumnTypeString,
				},
				{
					Name: "created_at",
					Type: rpc.ColumnTypeString,
				},
				{
					Name: "published_at",
					Type: rpc.ColumnTypeString,
				},
				{
					Name: "by",
					Type: rpc.ColumnTypeString,
				},
				{
					Name: "url",
					Type: rpc.ColumnTypeString,
				},
				{
					Name: "assets",
					Type: rpc.ColumnTypeString,
				},
			},
		}, nil
}

type releaseTable struct {
	client *github.Client
	db     *badger.DB
}

type releaseCursor struct {
	client *github.Client
	db     *badger.DB
	pageID int
}

// Return a slice of rows that will be returned to Anyquery and filtered.
// The second return value is true if the cursor has no more rows to return
//
// The constraints are used for optimization purposes to "pre-filter" the rows
// If the rows returned don't match the constraints, it's not an issue. Anyquery will filter them out
func (t *releaseCursor) Query(constraints rpc.QueryConstraint) ([][]interface{}, bool, error) {
	repository := retrieveArgString(constraints, 0)

	if repository == "" {
		return nil, true, fmt.Errorf("missing repository")
	}

	cacheKey := fmt.Sprintf("release-%d-%s", t.pageID, repository)

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

	// Retrieve the repositories from the API
	releases, _, err := t.client.Repositories.ListReleases(context.Background(), owner, name, &github.ListOptions{
		Page:    t.pageID,
		PerPage: 100,
	})

	if err != nil {
		return nil, true, fmt.Errorf("failed to retrieve releases: %v", err)
	}

	for _, release := range releases {
		rows = append(rows, []interface{}{
			release.GetID(),
			release.GetName(),
			release.GetTagName(),
			release.GetBody(),
			release.GetCreatedAt().Format(time.RFC3339),
			release.GetPublishedAt().Format(time.RFC3339),
			release.GetAuthor().GetLogin(),
			release.GetHTMLURL(),
			serializeJSON(release.Assets),
		})
	}

	// Save the repositories in the cache
	err = saveCache(t.db, cacheKey, rows)
	if err != nil {
		log.Printf("failed to save cache: %v", err)
	}

	t.pageID++

	return rows, len(rows) == 0, nil
}

// Create a new cursor that will be used to read rows
func (t *releaseTable) CreateReader() rpc.ReaderInterface {
	return &releaseCursor{
		t.client, t.db, 1,
	}
}

// A slice of rows to insert
func (t *releaseTable) Insert(rows [][]interface{}) error {
	return nil
}

// A slice of rows to update
// The first element of each row is the primary key
// while the rest are the values to update
// The primary key is therefore present twice
func (t *releaseTable) Update(rows [][]interface{}) error {
	return nil
}

// A slice of primary keys to delete
func (t *releaseTable) Delete(primaryKeys []interface{}) error {
	return nil
}

// A destructor to clean up resources
func (t *releaseTable) Close() error {
	return nil
}
