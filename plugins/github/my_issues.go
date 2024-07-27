package main

import (
	"context"
	"fmt"
	"time"

	"github.com/dgraph-io/badger/v4"
	"github.com/google/go-github/v63/github"
	"github.com/julien040/anyquery/rpc"
)

// A constructor to create a new table instance
// This function is called everytime a new connection is made to the plugin
//
// It should return a new table instance, the database schema and if there is an error
func my_issuesCreator(args rpc.TableCreatorArgs) (rpc.Table, *rpc.DatabaseSchema, error) {
	// Request a connection
	client, token, err := getClient(args)
	if err != nil {
		return nil, nil, err
	}

	// Open the database
	db, err := openDatabase("my_issues", token)
	if err != nil {
		return nil, nil, err
	}
	return &my_issuesTable{
			client, db,
		}, &rpc.DatabaseSchema{
			Columns: []rpc.DatabaseSchemaColumn{
				{
					Name:        "filter",
					Type:        rpc.ColumnTypeString,
					IsParameter: true,
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
					Name: "by",
					Type: rpc.ColumnTypeString,
				},
				{
					Name: "user_url",
					Type: rpc.ColumnTypeString,
				},
				{
					Name: "title",
					Type: rpc.ColumnTypeString,
				},
				{
					Name: "state",
					Type: rpc.ColumnTypeInt,
				},
				{
					Name: "locked",
					Type: rpc.ColumnTypeBool,
				},
				{
					Name: "author_association",
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
					Name: "comments",
					Type: rpc.ColumnTypeInt,
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
					Name: "closed_at",
					Type: rpc.ColumnTypeString,
				},
				{
					Name: "closed_by",
					Type: rpc.ColumnTypeString,
				},
				{
					Name: "is_pull_request",
					Type: rpc.ColumnTypeString,
				},
				{
					Name: "repository",
					Type: rpc.ColumnTypeString,
				},
			},
		}, nil
}

type my_issuesTable struct {
	client *github.Client
	db     *badger.DB
}

type my_issuesCursor struct {
	client *github.Client
	db     *badger.DB
	pageID int
}

// Return a slice of rows that will be returned to Anyquery and filtered.
// The second return value is true if the cursor has no more rows to return
//
// The constraints are used for optimization purposes to "pre-filter" the rows
// If the rows returned don't match the constraints, it's not an issue. Anyquery will filter them out
func (t *my_issuesCursor) Query(constraints rpc.QueryConstraint) ([][]interface{}, bool, error) {
	// Extract the filter from the constraints
	filter := ""
	var ok bool
	for _, constraint := range constraints.Columns {
		if constraint.ColumnID == 0 {
			filter, ok = constraint.Value.(string)
			if !ok {
				return nil, true, fmt.Errorf("filter is not a string")
			}
		}
	}
	if filter == "" {
		filter = "all"
	}

	cacheKey := fmt.Sprintf("issues-assigned-%d-%s", t.pageID, filter)

	// Check the cache
	rows := [][]interface{}{}
	err := loadCache(t.db, cacheKey, &rows)
	if err == nil {
		t.pageID++
		return rows, len(rows) == 0, nil
	}

	// Get the issues from the API
	issues, _, err := t.client.Issues.List(context.Background(), true, &github.IssueListOptions{
		State:  "all",
		Filter: filter,
		ListOptions: github.ListOptions{
			Page:    t.pageID,
			PerPage: 100,
		},
	})

	if err != nil {
		return nil, true, fmt.Errorf("failed to get issues: %w", err)
	}

	// Convert the issues to rows
	for _, issue := range issues {
		assignees := []string{}
		for _, assignee := range issue.Assignees {
			assignees = append(assignees, assignee.GetLogin())
		}

		labels := []string{}
		for _, label := range issue.Labels {
			labels = append(labels, label.GetName())
		}

		rows = append(rows, []interface{}{
			issue.GetID(),
			issue.GetNumber(),
			issue.GetUser().GetLogin(),
			issue.GetUser().GetHTMLURL(),
			issue.GetTitle(),
			issue.GetState(),
			issue.GetLocked(),
			issue.GetAuthorAssociation(),
			assignees,
			labels,
			issue.GetComments(),
			issue.GetCreatedAt().Format(time.RFC3339),
			issue.GetUpdatedAt().Format(time.RFC3339),
			issue.GetClosedAt().Format(time.RFC3339),
			issue.GetClosedBy().GetLogin(),
			issue.IsPullRequest(),
			issue.GetRepository().GetFullName(),
		})
	}

	// Save the rows in the cache
	err = saveCache(t.db, cacheKey, rows)
	if err != nil {
		return nil, true, fmt.Errorf("failed to save cache: %w", err)
	}

	t.pageID++

	return rows, len(rows) == 0, nil
}

// Create a new cursor that will be used to read rows
func (t *my_issuesTable) CreateReader() rpc.ReaderInterface {
	return &my_issuesCursor{
		client: t.client,
		db:     t.db,
		pageID: 1,
	}
}

// A slice of rows to insert
func (t *my_issuesTable) Insert(rows [][]interface{}) error {
	return nil
}

// A slice of rows to update
// The first element of each row is the primary key
// while the rest are the values to update
// The primary key is therefore present twice
func (t *my_issuesTable) Update(rows [][]interface{}) error {
	return nil
}

// A slice of primary keys to delete
func (t *my_issuesTable) Delete(primaryKeys []interface{}) error {
	return nil
}

// A destructor to clean up resources
func (t *my_issuesTable) Close() error {
	return nil
}
