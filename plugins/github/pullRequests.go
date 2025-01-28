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
func pullRequestsCreator(args rpc.TableCreatorArgs) (rpc.Table, *rpc.DatabaseSchema, error) {
	// Request a connection
	client, token, err := getClient(args)
	if err != nil {
		return nil, nil, err
	}

	// Open the database
	db, err := openDatabase("pr", token)
	if err != nil {
		return nil, nil, err
	}
	return &pullRequestsTable{
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
					Name:        "id",
					Type:        rpc.ColumnTypeInt,
					Description: "The ID of the pull request",
				},
				{
					Name:        "number",
					Type:        rpc.ColumnTypeInt,
					Description: "The number ID of the pull request. In https://github.com/owner/name/pull/16, the number ID is 16",
				},
				{
					Name:        "title",
					Type:        rpc.ColumnTypeString,
					Description: "The title of the pull request",
				},
				{
					Name:        "body",
					Type:        rpc.ColumnTypeString,
					Description: "The markdown body of the pull request",
				},
				{
					Name:        "state",
					Type:        rpc.ColumnTypeString,
					Description: "The state of the pull request (open, closed, merged)",
				},
				{
					Name:        "by",
					Type:        rpc.ColumnTypeString,
					Description: "The username of the user who opened the pull request",
				},
				{
					Name:        "assignees",
					Type:        rpc.ColumnTypeJSON,
					Description: "A JSON array of assignees",
				},
				{
					Name:        "labels",
					Type:        rpc.ColumnTypeJSON,
					Description: "A JSON array of labels",
				},
				{
					Name: "closed_at",
					Type: rpc.ColumnTypeDateTime,
				},
				{
					Name: "created_at",
					Type: rpc.ColumnTypeDateTime,
				},
				{
					Name: "updated_at",
					Type: rpc.ColumnTypeDateTime,
				},
				{
					Name:        "merged_at",
					Type:        rpc.ColumnTypeDateTime,
					Description: "The date and time the pull request was merged (RFC3339 format). Can be null if the pull request is open or closed",
				},
				{
					Name:        "merged_by",
					Type:        rpc.ColumnTypeString,
					Description: "The username of the user who merged the pull request. Can be null if the pull request is open or closed",
				},
				{
					Name:        "url",
					Type:        rpc.ColumnTypeString,
					Description: "The URL to see the pull request on GitHub",
				},
			},
		}, nil
}

type pullRequestsTable struct {
	client *github.Client
	db     *badger.DB
}

type pullRequestsCursor struct {
	client *github.Client
	db     *badger.DB
	pageID int
}

// Return a slice of rows that will be returned to Anyquery and filtered.
// The second return value is true if the cursor has no more rows to return
//
// The constraints are used for optimization purposes to "pre-filter" the rows
// If the rows returned don't match the constraints, it's not an issue. Anyquery will filter them out
func (t *pullRequestsCursor) Query(constraints rpc.QueryConstraint) ([][]interface{}, bool, error) {
	repository := retrieveArgString(constraints, 0)

	if repository == "" {
		return nil, true, fmt.Errorf("missing repository")
	}

	cacheKey := fmt.Sprintf("pr-%d-%s", t.pageID, repository)

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

	// Retrieve the commits
	pr, _, err := t.client.PullRequests.List(context.Background(), owner, name, &github.PullRequestListOptions{
		ListOptions: github.ListOptions{
			Page:    t.pageID,
			PerPage: 100,
		},
		State: "all",
	})

	if err != nil {
		return nil, true, fmt.Errorf("failed to retrieve pull requests: %w", err)
	}

	for _, pr := range pr {
		assignees := []string{}
		for _, assignee := range pr.Assignees {
			assignees = append(assignees, assignee.GetLogin())
		}
		labels := []string{}
		for _, label := range pr.Labels {
			labels = append(labels, label.GetName())
		}
		rows = append(rows, []interface{}{
			pr.GetID(),
			pr.GetNumber(),
			pr.GetTitle(),
			pr.GetBody(),
			pr.GetState(),
			pr.GetUser().GetLogin(),
			assignees,
			labels,
			pr.GetClosedAt().Format(time.RFC3339),
			pr.GetCreatedAt().Format(time.RFC3339),
			pr.GetUpdatedAt().Format(time.RFC3339),
			pr.GetMergedAt().Format(time.RFC3339),
			pr.GetMergedBy().GetLogin(),
			pr.GetHTMLURL(),
		})
	}

	// Save the cache
	err = saveCache(t.db, cacheKey, rows)
	if err != nil {
		log.Printf("failed to save cache: %v", err)
	}

	t.pageID++

	return rows, len(rows) == 0, nil
}

// Create a new cursor that will be used to read rows
func (t *pullRequestsTable) CreateReader() rpc.ReaderInterface {
	return &pullRequestsCursor{
		client: t.client,
		db:     t.db,
		pageID: 1,
	}
}

// A destructor to clean up resources
func (t *pullRequestsTable) Close() error {
	return nil
}
