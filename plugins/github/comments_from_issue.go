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
func comments_from_issueCreator(args rpc.TableCreatorArgs) (rpc.Table, *rpc.DatabaseSchema, error) {
	// Request a connection
	client, token, err := getClient(args)
	if err != nil {
		return nil, nil, err
	}

	// Open the database
	db, err := openDatabase("issues_comments", token)
	if err != nil {
		return nil, nil, err
	}
	return &comments_from_issueTable{client, db}, &rpc.DatabaseSchema{
		PrimaryKey: 2,
		Columns: []rpc.DatabaseSchemaColumn{
			{
				Name:        "repository",
				Type:        rpc.ColumnTypeString,
				IsParameter: true,
				IsRequired:  true,
				Description: "The repository in the format owner/name",
			},
			{
				Name:        "issue",
				Type:        rpc.ColumnTypeInt,
				IsParameter: true,
				IsRequired:  true,
				Description: "The ID of the issue. Can be found in the URL of the issue. In https://github.com/julien040/anyquery/issues/16, the issue ID is 16",
			},
			{
				Name:        "id",
				Type:        rpc.ColumnTypeString,
				Description: "The ID of the comment",
			},
			{
				Name:        "body",
				Type:        rpc.ColumnTypeString,
				Description: "The markdown body of the comment",
			},
			{
				Name:        "by",
				Type:        rpc.ColumnTypeString,
				Description: "The username of the user who posted the comment",
			},
			{
				Name:        "user_url",
				Type:        rpc.ColumnTypeString,
				Description: "The URL to the user's profile",
			},
			{
				Name:        "created_at",
				Type:        rpc.ColumnTypeDateTime,
				Description: "The date and time the comment was created (RFC3339 format)",
			},
			{
				Name:        "updated_at",
				Type:        rpc.ColumnTypeDateTime,
				Description: "The date and time the comment was last updated (RFC3339 format)",
			},
			{
				Name:        "author_association",
				Type:        rpc.ColumnTypeString,
				Description: "The association of the user who posted the comment to the repository. One of OWNER, COLLABORATOR, CONTRIBUTOR, MEMBER, FIRST_TIME_CONTRIBUTOR, FIRST_TIMER, NONE",
			},
			{
				Name:        "reactions",
				Type:        rpc.ColumnTypeJSON,
				Description: "A JSON object containing the reactions for each reaction type (total_count, +1, -1, laugh, confused, heart, hooray, rocket, eyes)",
			},
			{
				Name:        "url",
				Type:        rpc.ColumnTypeString,
				Description: "The URL to see the comment on GitHub",
			},
		},
	}, nil
}

type comments_from_issueTable struct {
	client *github.Client
	db     *badger.DB
}

type comments_from_issueCursor struct {
	client *github.Client
	db     *badger.DB
	pageID int
}

// Return a slice of rows that will be returned to Anyquery and filtered.
// The second return value is true if the cursor has no more rows to return
//
// The constraints are used for optimization purposes to "pre-filter" the rows
// If the rows returned don't match the constraints, it's not an issue. Anyquery will filter them out
func (t *comments_from_issueCursor) Query(constraints rpc.QueryConstraint) ([][]interface{}, bool, error) {
	// Get the repository and comment_id from the constraints
	repository := retrieveArgString(constraints, 0)
	if repository == "" {
		return nil, true, fmt.Errorf("repository is required")
	}

	splitted := strings.Split(repository, "/")
	if len(splitted) != 2 {
		return nil, true, fmt.Errorf("repository must be in the format owner/repo")
	}
	owner := splitted[0]
	repository = splitted[1]

	commentID := -1
	for _, c := range constraints.Columns {
		if c.ColumnID == 1 {
			switch c.Value.(type) {
			case int:
				commentID = c.Value.(int)
			case int64:
				commentID = int(c.Value.(int64))
			case string:
				fmt.Sscanf(c.Value.(string), "%d", &commentID)
			}
		}
	}

	if commentID == -1 {
		return nil, true, fmt.Errorf("comment_id is required")
	}

	cacheKey := fmt.Sprintf("comments-%s-%d-%d", repository, commentID, t.pageID)

	// Check the cache
	rows := [][]interface{}{}
	err := loadCache(t.db, cacheKey, &rows)
	if err == nil {
		t.pageID++
		return rows, len(rows) == 0, nil
	}

	// Get the comments from the API
	comments, _, err := t.client.Issues.ListComments(context.Background(), owner, repository, commentID, &github.IssueListCommentsOptions{
		ListOptions: github.ListOptions{
			Page:    t.pageID,
			PerPage: 100,
		},
	})

	if err != nil {
		return nil, true, fmt.Errorf("failed to get comments: %w", err)
	}

	// Convert the comments to rows
	for _, comment := range comments {
		rows = append(rows, []interface{}{
			comment.GetID(),
			comment.GetBody(),
			comment.GetUser().GetLogin(),
			comment.GetUser().GetHTMLURL(),
			comment.GetCreatedAt().Format(time.RFC3339),
			comment.GetUpdatedAt().Format(time.RFC3339),
			comment.GetAuthorAssociation(),
			serializeJSON(comment.GetReactions()),
			comment.GetHTMLURL(),
		})
	}

	// Save the rows in the cache
	err = saveCache(t.db, cacheKey, rows)
	if err != nil {
		log.Printf("failed to save cache: %v", err)
	}

	return rows, len(rows) == 0, nil
}

// Create a new cursor that will be used to read rows
func (t *comments_from_issueTable) CreateReader() rpc.ReaderInterface {
	return &comments_from_issueCursor{
		client: t.client,
		db:     t.db,
		pageID: 1,
	}
}

// A destructor to clean up resources
func (t *comments_from_issueTable) Close() error {
	return nil
}
