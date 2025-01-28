package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/dgraph-io/badger/v4"
	"github.com/google/go-github/v63/github"
	"github.com/julien040/anyquery/rpc"
)

// A constructor to create a new table instance
// This function is called everytime a new connection is made to the plugin
//
// It should return a new table instance, the database schema and if there is an error
func starsCreator(args rpc.TableCreatorArgs) (rpc.Table, *rpc.DatabaseSchema, error) {
	// Request a connection
	client, token, err := getClient(args)
	if err != nil {
		return nil, nil, err
	}

	columns := []rpc.DatabaseSchemaColumn{
		{
			Name:        "user",
			Type:        rpc.ColumnTypeString,
			IsParameter: true,
			IsRequired:  true,
			Description: "The user to get the starred repositories from",
		},
		{
			Name:        "starred_at",
			Type:        rpc.ColumnTypeDateTime,
			Description: "The date when the user starred the repository (RFC3339)",
		},
	}
	columns = append(columns, repositorySchema...)

	// Open the database
	db, err := openDatabase("star_from_user", token)
	if err != nil {
		return nil, nil, err
	}
	return &starsTable{
			client: client,
			db:     db,
		}, &rpc.DatabaseSchema{
			PrimaryKey: 1,
			Columns:    columns,
		}, nil
}

type starsTable struct {
	client *github.Client
	db     *badger.DB
}

type starsCursor struct {
	client *github.Client
	db     *badger.DB
	pageID int
}

// Return a slice of rows that will be returned to Anyquery and filtered.
// The second return value is true if the cursor has no more rows to return
//
// The constraints are used for optimization purposes to "pre-filter" the rows
// If the rows returned don't match the constraints, it's not an issue. Anyquery will filter them out
func (t *starsCursor) Query(constraints rpc.QueryConstraint) ([][]interface{}, bool, error) {
	user := retrieveArgString(constraints, 0)

	if user == "" {
		return nil, true, fmt.Errorf("missing user")
	}

	cacheKey := fmt.Sprintf("star-%d-%s", t.pageID, user)

	// Retrieve the repositories from the cache
	rows := [][]interface{}{}
	err := loadCache(t.db, cacheKey, &rows)
	if err == nil {
		t.pageID++
		return rows, len(rows) == 0, nil
	}

	// Retrieve the repositories
	repos, _, err := t.client.Activity.ListStarred(context.Background(), user, &github.ActivityListStarredOptions{
		ListOptions: github.ListOptions{
			Page:    t.pageID,
			PerPage: 100,
		},
	})
	if err != nil {
		return nil, true, fmt.Errorf("failed to get starred repositories: %w", err)
	}

	// Convert the repositories to rows
	for _, repoStar := range repos {
		repo := repoStar.GetRepository()
		webhookCommitSignOff := false
		if repo.WebCommitSignoffRequired != nil && *repo.WebCommitSignoffRequired {
			webhookCommitSignOff = true
		}

		rows = append(rows, []interface{}{
			repoStar.GetStarredAt().Format(time.RFC3339),
			repo.GetID(),
			repo.GetNodeID(),
			repo.GetOwner().GetLogin(),
			repo.GetName(),
			repo.GetFullName(),
			repo.GetDescription(),
			repo.GetHomepage(),
			repo.GetDefaultBranch(),
			repo.GetCreatedAt().Format(time.RFC3339),
			repo.GetPushedAt().Format(time.RFC3339),
			repo.GetUpdatedAt().Format(time.RFC3339),
			repo.GetHTMLURL(),
			repo.GetCloneURL(),
			repo.GetGitURL(),
			repo.GetMirrorURL(),
			repo.GetSSHURL(),
			repo.GetLanguage(),
			repo.GetFork(),
			repo.GetForksCount(),
			repo.GetNetworkCount(),
			repo.GetOpenIssuesCount(),
			repo.GetStargazersCount(),
			repo.GetSubscribersCount(),
			repo.GetSize(),
			repo.GetAllowRebaseMerge(),
			repo.GetAllowUpdateBranch(),
			repo.GetAllowSquashMerge(),
			repo.GetAllowMergeCommit(),
			repo.GetAllowAutoMerge(),
			repo.GetAllowForking(),
			webhookCommitSignOff,
			repo.GetDeleteBranchOnMerge(),
			repo.Topics,
			serializeJSON(repo.GetCustomProperties()),
			repo.GetArchived(),
			repo.GetDisabled(),
			repo.GetVisibility(),
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
func (t *starsTable) CreateReader() rpc.ReaderInterface {
	return &starsCursor{
		client: t.client,
		db:     t.db,
		pageID: 1,
	}
}

// A destructor to clean up resources
func (t *starsTable) Close() error {
	return nil
}
