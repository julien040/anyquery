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
func contributorsCreator(args rpc.TableCreatorArgs) (rpc.Table, *rpc.DatabaseSchema, error) {
	// Request a connection
	client, token, err := getClient(args)
	if err != nil {
		return nil, nil, err
	}

	// Open the database
	db, err := openDatabase("contributors", token)
	if err != nil {
		return nil, nil, err
	}
	return &contributorsTable{
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
					Description: "The username of the contributor",
				},
				{
					Name: "contributor_url",
					Type: rpc.ColumnTypeString,
				},
				{
					Name:        "additions",
					Type:        rpc.ColumnTypeInt,
					Description: "The number of lines added by the contributor",
				},
				{
					Name:        "deletions",
					Type:        rpc.ColumnTypeInt,
					Description: "The number of lines deleted by the contributor",
				},
				{
					Name:        "commits",
					Type:        rpc.ColumnTypeInt,
					Description: "The number of commits made by the contributor",
				},
			},
		}, nil
}

type contributorsTable struct {
	client *github.Client
	db     *badger.DB
}

type contributorsCursor struct {
	client *github.Client
	db     *badger.DB
}

// Return a slice of rows that will be returned to Anyquery and filtered.
// The second return value is true if the cursor has no more rows to return
//
// The constraints are used for optimization purposes to "pre-filter" the rows
// If the rows returned don't match the constraints, it's not an issue. Anyquery will filter them out
func (t *contributorsCursor) Query(constraints rpc.QueryConstraint) ([][]interface{}, bool, error) {
	repository := retrieveArgString(constraints, 0)

	if repository == "" {
		return nil, true, fmt.Errorf("missing repository")
	}

	cacheKey := fmt.Sprintf("contributors-%s", repository)

	// Retrieve the repositories from the cache
	rows := [][]interface{}{}
	err := loadCache(t.db, cacheKey, &rows)
	if err == nil {
		return rows, true, nil
	}

	splitted := strings.Split(repository, "/")
	if len(splitted) != 2 {
		return nil, true, fmt.Errorf("repository must be in the format owner/name")
	}

	owner := splitted[0]
	name := splitted[1]

	// Retrieve the commits
	stats, res, err := t.client.Repositories.ListContributorsStats(context.Background(), owner, name)
	if res.StatusCode == 202 {
		// We wait for GitHub to compute the stats and then we retry
		time.Sleep(2 * time.Second)
		stats, _, err = t.client.Repositories.ListContributorsStats(context.Background(), owner, name)
	}

	if err != nil {
		return nil, true, fmt.Errorf("failed to retrieve contributors: %v", err)
	}

	for _, stat := range stats {
		var additions, deletions, commits int64
		for _, week := range stat.Weeks {
			additions += int64(week.GetAdditions())
			deletions += int64(week.GetDeletions())
			commits += int64(week.GetCommits())
		}
		rows = append(rows, []interface{}{
			stat.Author.GetLogin(),
			stat.Author.GetHTMLURL(),
			additions,
			deletions,
			commits,
		})
	}

	err = saveCache(t.db, cacheKey, rows)
	if err != nil {
		log.Printf("failed to save cache: %v", err)
	}

	return rows, true, nil
}

// Create a new cursor that will be used to read rows
func (t *contributorsTable) CreateReader() rpc.ReaderInterface {
	return &contributorsCursor{
		client: t.client,
		db:     t.db,
	}
}

// A destructor to clean up resources
func (t *contributorsTable) Close() error {
	return nil
}
