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
func commitsCreator(args rpc.TableCreatorArgs) (rpc.Table, *rpc.DatabaseSchema, error) {
	// Request a connection
	client, token, err := getClient(args)
	if err != nil {
		return nil, nil, err
	}

	// Open the database
	db, err := openDatabase("commits", token)
	if err != nil {
		return nil, nil, err
	}

	return &commitsTable{
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
					Name: "sha",
					Type: rpc.ColumnTypeString,
				},
				{
					Name: "committer",
					Type: rpc.ColumnTypeString,
				},
				{
					Name: "committer_email",
					Type: rpc.ColumnTypeString,
				},
				{
					Name: "committer_date",
					Type: rpc.ColumnTypeString,
				},
				{
					Name: "author",
					Type: rpc.ColumnTypeString,
				},
				{
					Name: "author_email",
					Type: rpc.ColumnTypeString,
				},
				{
					Name: "author_date",
					Type: rpc.ColumnTypeString,
				},
				{
					Name: "message",
					Type: rpc.ColumnTypeString,
				},
				{
					Name: "html_url",
					Type: rpc.ColumnTypeString,
				},
			},
		}, nil
}

type commitsTable struct {
	client *github.Client
	db     *badger.DB
}

type commitsCursor struct {
	client *github.Client
	db     *badger.DB
	pageID int
}

// Return a slice of rows that will be returned to Anyquery and filtered.
// The second return value is true if the cursor has no more rows to return
//
// The constraints are used for optimization purposes to "pre-filter" the rows
// If the rows returned don't match the constraints, it's not an issue. Anyquery will filter them out
func (t *commitsCursor) Query(constraints rpc.QueryConstraint) ([][]interface{}, bool, error) {
	repository := retrieveArgString(constraints, 0)

	if repository == "" {
		return nil, true, fmt.Errorf("missing repository")
	}

	cacheKey := fmt.Sprintf("commits-%d-%s", t.pageID, repository)

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
	commits, _, err := t.client.Repositories.ListCommits(context.Background(), owner, name, &github.CommitsListOptions{
		ListOptions: github.ListOptions{
			Page:    t.pageID,
			PerPage: 100,
		},
	})

	if err != nil {
		return nil, true, fmt.Errorf("error with github api: %w", err)
	}

	for _, commit := range commits {
		var committerDate, authorDate, message, committerEmail, authorEmail string
		if commit.Commit != nil {
			gitCommit := commit.Commit
			if gitCommit.Committer != nil {
				committerDate = gitCommit.Committer.Date.Format(time.RFC3339)
				if gitCommit.Committer.Email != nil {
					committerEmail = *gitCommit.Committer.Email
				}
			}
			if gitCommit.Author != nil {
				authorDate = gitCommit.Author.Date.Format(time.RFC3339)
				if gitCommit.Author.Email != nil {
					authorEmail = *gitCommit.Author.Email
				}
			}

			message = gitCommit.GetMessage()
		}
		row := []interface{}{
			commit.GetSHA(),
			commit.GetCommitter().GetLogin(),
			committerEmail,
			committerDate,
			commit.GetAuthor().GetLogin(),
			authorEmail,
			authorDate,
			message,
			commit.GetHTMLURL(),
		}

		rows = append(rows, row)
	}

	// Save the cache
	err = saveCache(t.db, cacheKey, rows)
	if err != nil {
		log.Printf("Failed to save cache: %v", err)
	}

	t.pageID++

	return rows, len(rows) == 0, nil
}

// Create a new cursor that will be used to read rows
func (t *commitsTable) CreateReader() rpc.ReaderInterface {
	return &commitsCursor{
		client: t.client,
		db:     t.db,
		pageID: 1,
	}
}

// A slice of rows to insert
func (t *commitsTable) Insert(rows [][]interface{}) error {
	return nil
}

// A slice of rows to update
// The first element of each row is the primary key
// while the rest are the values to update
// The primary key is therefore present twice
func (t *commitsTable) Update(rows [][]interface{}) error {
	return nil
}

// A slice of primary keys to delete
func (t *commitsTable) Delete(primaryKeys []interface{}) error {
	return nil
}

// A destructor to clean up resources
func (t *commitsTable) Close() error {
	return nil
}
