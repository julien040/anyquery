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
					Description: "The repository in the format owner/name",
				},
				{
					Name:        "id",
					Type:        rpc.ColumnTypeInt,
					Description: "The ID of the issue",
				},
				{
					Name:        "number",
					Type:        rpc.ColumnTypeInt,
					Description: "The number ID of the issue",
				},
				{
					Name:        "title",
					Type:        rpc.ColumnTypeString,
					Description: "The title of the issue",
				},
				{
					Name:        "body",
					Type:        rpc.ColumnTypeString,
					Description: "The markdown body of the issue",
				},
				{
					Name:        "state",
					Type:        rpc.ColumnTypeString,
					Description: "The state of the issue (open, closed)",
				},
				{
					Name:        "state_reason",
					Type:        rpc.ColumnTypeString,
					Description: "The reason why the issue is in the current state",
				},
				{
					Name:        "by",
					Type:        rpc.ColumnTypeString,
					Description: "The username of the user who posted the issue",
				},
				{
					Name:        "assignees",
					Type:        rpc.ColumnTypeJSON,
					Description: "A JSON array of the assignees of the issue",
				},
				{
					Name:        "labels",
					Type:        rpc.ColumnTypeString,
					Description: "A JSON array of the labels of the issue",
				},
				{
					Name:        "closed_at",
					Type:        rpc.ColumnTypeDateTime,
					Description: "The date and time the issue was closed (RFC3339 format). Can be null if the issue is open",
				},
				{
					Name:        "closed_by",
					Type:        rpc.ColumnTypeString,
					Description: "The username of the user who closed the issue",
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
					Name:        "url",
					Type:        rpc.ColumnTypeString,
					Description: "The URL to see the issue on GitHub",
				},
				{
					Name:        "reactions",
					Type:        rpc.ColumnTypeJSON,
					Description: "A JSON object containing the reactions for each reaction type (total_count, +1, -1, laugh, confused, heart, hooray, rocket, eyes)",
				},
				{
					Name: "draft",
					Type: rpc.ColumnTypeBool,
				},
				{
					Name: "locked",
					Type: rpc.ColumnTypeBool,
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

// A destructor to clean up resources
func (t *issuesTable) Close() error {
	return nil
}
