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
func my_starsCreator(args rpc.TableCreatorArgs) (rpc.Table, *rpc.DatabaseSchema, error) {
	// Request a connection
	client, token, err := getClient(args)
	if err != nil {
		return nil, nil, err
	}

	columns := []rpc.DatabaseSchemaColumn{
		{
			Name:        "starred_at",
			Type:        rpc.ColumnTypeString,
			Description: "The date the repository was starred",
		},
	}
	columns = append(columns, repositorySchema...)

	// Open the database
	db, err := openDatabase("my_star", token)
	if err != nil {
		return nil, nil, err
	}
	return &my_starsTable{client: client,
			db: db,
		}, &rpc.DatabaseSchema{
			PrimaryKey: 1,
			Columns:    columns,
		}, nil
}

type my_starsTable struct {
	client *github.Client
	db     *badger.DB
}

type my_starsCursor struct {
	client *github.Client
	db     *badger.DB
	pageID int
}

// Return a slice of rows that will be returned to Anyquery and filtered.
// The second return value is true if the cursor has no more rows to return
//
// The constraints are used for optimization purposes to "pre-filter" the rows
// If the rows returned don't match the constraints, it's not an issue. Anyquery will filter them out
func (t *my_starsCursor) Query(constraints rpc.QueryConstraint) ([][]interface{}, bool, error) {
	cacheKey := fmt.Sprintf("star-%d", t.pageID)

	// Retrieve the repositories from the cache
	rows := [][]interface{}{}
	err := loadCache(t.db, cacheKey, &rows)
	if err == nil {
		t.pageID++
		return rows, len(rows) == 0, nil
	}

	// Retrieve the repositories
	repos, _, err := t.client.Activity.ListStarred(context.Background(), "", &github.ActivityListStarredOptions{
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
func (t *my_starsTable) CreateReader() rpc.ReaderInterface {
	return &my_starsCursor{
		client: t.client,
		db:     t.db,
	}
}

// A destructor to clean up resources
func (t *my_starsTable) Close() error {
	return nil
}
