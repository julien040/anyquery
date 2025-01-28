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
					Description: "The filter to apply to the issues. Can be one of: assigned, created, mentioned, subscribed, all. If this parameter is not provided, all issues are returned.",
				},
				{
					Name:        "id",
					Type:        rpc.ColumnTypeInt,
					Description: "The ID of the issue",
				},
				{
					Name:        "number",
					Type:        rpc.ColumnTypeInt,
					Description: "The number ID of the issue. Will be found in https://github.com/owner/repo/issues/number",
				},
				{
					Name:        "by",
					Type:        rpc.ColumnTypeString,
					Description: "The username of the user who created the issue",
				},
				{
					Name:        "user_url",
					Type:        rpc.ColumnTypeString,
					Description: "The URL of the user who created the issue",
				},
				{
					Name:        "title",
					Type:        rpc.ColumnTypeString,
					Description: "The title of the issue",
				},
				{
					Name:        "state",
					Type:        rpc.ColumnTypeInt,
					Description: "The state of the issue (open, closed)",
				},
				{
					Name:        "locked",
					Type:        rpc.ColumnTypeBool,
					Description: "Whether the issue is locked",
				},
				{
					Name:        "author_association",
					Type:        rpc.ColumnTypeString,
					Description: "The author association of the user who created the issue. Can be one of: OWNER, COLLABORATOR, CONTRIBUTOR, MEMBER",
				},
				{
					Name:        "assignees",
					Type:        rpc.ColumnTypeJSON,
					Description: "A JSON array of the assignees of the issue",
				},
				{
					Name:        "labels",
					Type:        rpc.ColumnTypeJSON,
					Description: "A JSON array of the labels of the issue",
				},
				{
					Name:        "comments",
					Type:        rpc.ColumnTypeInt,
					Description: "The number of comments on the issue",
				},
				{
					Name:        "created_at",
					Type:        rpc.ColumnTypeDateTime,
					Description: "The date and time the issue was created (RFC3339 format)",
				},
				{
					Name:        "updated_at",
					Type:        rpc.ColumnTypeDateTime,
					Description: "The date and time the issue was last updated (RFC3339 format)",
				},
				{
					Name:        "closed_at",
					Type:        rpc.ColumnTypeDateTime,
					Description: "The date and time the issue was closed (RFC3339 format). Can be null if the issue is open",
				},
				{
					Name:        "closed_by",
					Type:        rpc.ColumnTypeString,
					Description: "The username of the user who closed the issue. Can be null if the issue is open",
				},
				{
					Name:        "is_pull_request",
					Type:        rpc.ColumnTypeBool,
					Description: "Whether the issue is a pull request",
				},
				{
					Name:        "repository",
					Type:        rpc.ColumnTypeString,
					Description: "The repository in the format owner/name",
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

// A destructor to clean up resources
func (t *my_issuesTable) Close() error {
	return nil
}
