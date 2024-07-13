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
func issuesCreator(args rpc.TableCreatorArgs) (rpc.Table, *rpc.DatabaseSchema, error) {
	// Request a connection
	client, token, err := getClient(args)
	if err != nil {
		return nil, nil, err
	}

	// Open the database
	db, err := openDatabase("issues", token)
	if err != nil {
		return nil, nil, err
	}
	return &issuesTable{
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
					Name: "number",
					Type: rpc.ColumnTypeInt,
				},
				{
					Name: "title",
					Type: rpc.ColumnTypeString,
				},
				{
					Name: "body",
					Type: rpc.ColumnTypeString,
				},
				{
					Name: "state",
					Type: rpc.ColumnTypeString,
				},
				{
					Name: "state_reason",
					Type: rpc.ColumnTypeString,
				},
				{
					Name: "by",
					Type: rpc.ColumnTypeString,
				},
				{
					Name: "assignees",
					Type: rpc.ColumnTypeString,
				},
				{
					Name: "labels",
					Type: rpc.ColumnTypeString,
				},
				{
					Name: "closed_at",
					Type: rpc.ColumnTypeString,
				},
				{
					Name: "closed_by",
					Type: rpc.ColumnTypeString,
				},
				{
					Name: "created_at",
					Type: rpc.ColumnTypeString,
				},
				{
					Name: "updated_at",
					Type: rpc.ColumnTypeString,
				},
				{
					Name: "url",
					Type: rpc.ColumnTypeString,
				},
				{
					Name: "reactions",
					Type: rpc.ColumnTypeString,
				},
				{
					Name: "draft",
					Type: rpc.ColumnTypeInt,
				},
				{
					Name: "locked",
					Type: rpc.ColumnTypeInt,
				},
			},
		}, nil
}

type issuesTable struct {
	client *github.Client
	db     *badger.DB
}

type issuesCursor struct {
	client *github.Client
	db     *badger.DB
	pageID int
}

// Return a slice of rows that will be returned to Anyquery and filtered.
// The second return value is true if the cursor has no more rows to return
//
// The constraints are used for optimization purposes to "pre-filter" the rows
// If the rows returned don't match the constraints, it's not an issue. Anyquery will filter them out
func (t *issuesCursor) Query(constraints rpc.QueryConstraint) ([][]interface{}, bool, error) {
	repository := retrieveArgString(constraints, 0)

	if repository == "" {
		return nil, true, fmt.Errorf("missing repository")
	}

	cacheKey := fmt.Sprintf("issues-%d-%s", t.pageID, repository)

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
	issues, res, err := t.client.Issues.ListByRepo(context.Background(), owner, name, &github.IssueListByRepoOptions{
		ListOptions: github.ListOptions{
			Page:    t.pageID,
			PerPage: 100,
		},
		State: "all",
	})

	if res.StatusCode != 200 {
		return nil, true, fmt.Errorf("failed to retrieve issues: %v", res.Status)
	}

	if err != nil {
		return nil, true, fmt.Errorf("failed to retrieve issues: %v", err)
	}

	for _, issue := range issues {
		labels := []string{}
		for _, label := range issue.Labels {
			labels = append(labels, label.GetName())
		}

		assignees := []string{}
		for _, assignee := range issue.Assignees {
			assignees = append(assignees, assignee.GetLogin())
		}

		if issue.PullRequestLinks != nil {
			// This is a pull request
			continue
		}

		rows = append(rows, []interface{}{
			issue.GetID(),
			issue.GetNumber(),
			issue.GetTitle(),
			issue.GetBody(),
			issue.GetState(),
			issue.GetStateReason(),
			issue.GetUser().GetLogin(),
			assignees, // anyquery will convert this to a string
			labels,    // anyquery will convert this to a string
			issue.GetClosedAt().Format(time.RFC3339),
			issue.GetClosedBy().GetLogin(),
			issue.GetCreatedAt().Format(time.RFC3339),
			issue.GetUpdatedAt().Format(time.RFC3339),
			issue.GetHTMLURL(),
			serializeJSON(issue.Reactions),
			issue.GetDraft(),
			issue.GetLocked(),
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
func (t *issuesTable) CreateReader() rpc.ReaderInterface {
	return &issuesCursor{
		client: t.client,
		db:     t.db,
		pageID: 1,
	}
}

// A slice of rows to insert
func (t *issuesTable) Insert(rows [][]interface{}) error {
	return nil
}

// A slice of rows to update
// The first element of each row is the primary key
// while the rest are the values to update
// The primary key is therefore present twice
func (t *issuesTable) Update(rows [][]interface{}) error {
	return nil
}

// A slice of primary keys to delete
func (t *issuesTable) Delete(primaryKeys []interface{}) error {
	return nil
}

// A destructor to clean up resources
func (t *issuesTable) Close() error {
	return nil
}
