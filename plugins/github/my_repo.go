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
func myRepoCreator(args rpc.TableCreatorArgs) (rpc.Table, *rpc.DatabaseSchema, error) {
	// Request a connection
	client, token, err := getClient(args)
	if err != nil {
		return nil, nil, err
	}

	// Open the database
	db, err := openDatabase("my_repo", token)
	if err != nil {
		return nil, nil, err
	}

	return &githubTable{
			client: client,
			db:     db,
		}, &rpc.DatabaseSchema{
			Columns: repositorySchema,
		}, nil
}

type githubTable struct {
	client *github.Client
	db     *badger.DB
}

type githubCursor struct {
	client *github.Client
	pageID int
	db     *badger.DB
}

// Return a slice of rows that will be returned to Anyquery and filtered.
// The second return value is true if the cursor has no more rows to return
//
// The constraints are used for optimization purposes to "pre-filter" the rows
// If the rows returned don't match the constraints, it's not an issue. Anyquery will filter them out
func (t *githubCursor) Query(constraints rpc.QueryConstraint) ([][]interface{}, bool, error) {
	visibility := retrieveArgString(constraints, 37)
	if visibility != "private" && visibility != "public" {
		visibility = ""
	}

	// Retrieve the repositories
	rows := [][]interface{}{}
	cacheKey := fmt.Sprintf("repositories-%d-%s", t.pageID, visibility)

	// Try to load the cache
	err := loadCache(t.db, cacheKey, &rows)
	if err == nil {
		t.pageID++
		return rows, len(rows) == 0, nil
	}

	// Otherwise, fetch the repositories
	repos, _, err := t.client.Repositories.ListByAuthenticatedUser(context.Background(), &github.RepositoryListByAuthenticatedUserOptions{
		Visibility: visibility,
		ListOptions: github.ListOptions{
			Page:    t.pageID,
			PerPage: 100,
		},
	})

	if err != nil {
		return nil, true, fmt.Errorf("failed to list repositories: %w", err)
	}

	// Serialize the custom properties
	rows = convertRepo(repos, rows)

	// Save the cache
	err = saveCache(t.db, cacheKey, rows)
	if err != nil {
		log.Printf("Failed to save cache: %v", err)
	}

	t.pageID++

	return rows, len(rows) == 0, nil
}

func convertRepo(repos []*github.Repository, rows [][]interface{}) [][]interface{} {
	for _, repo := range repos {
		webhookCommitSignOff := false
		if repo.WebCommitSignoffRequired != nil && *repo.WebCommitSignoffRequired {
			webhookCommitSignOff = true
		}

		rows = append(rows, []interface{}{
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
	return rows
}

// Create a new cursor that will be used to read rows
func (t *githubTable) CreateReader() rpc.ReaderInterface {
	return &githubCursor{
		client: t.client,
		pageID: 1,
		db:     t.db,
	}
}

// A destructor to clean up resources
func (t *githubTable) Close() error {
	return nil
}
