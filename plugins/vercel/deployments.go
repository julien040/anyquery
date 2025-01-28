package main

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"log"
	"time"

	"github.com/dgraph-io/badger/v4"
	"github.com/julien040/anyquery/rpc"
)

// A constructor to create a new table instance
// This function is called everytime a new connection is made to the plugin
//
// It should return a new table instance, the database schema and if there is an error
func deploymentsCreator(args rpc.TableCreatorArgs) (rpc.Table, *rpc.DatabaseSchema, error) {
	token, err := getToken(args.UserConfig)
	if err != nil {
		return nil, nil, err
	}

	db, err := openDatabase("deployments", token)
	if err != nil {
		return nil, nil, err
	}
	return &deploymentsTable{
			token: token,
			db:    db,
		}, &rpc.DatabaseSchema{
			Columns: []rpc.DatabaseSchemaColumn{
				{
					Name:        "project_id",
					Type:        rpc.ColumnTypeString,
					IsParameter: true,
					Description: "The ID of the project. In https://vercel.com/samuel-project/samuel-app, the project ID is samuel-app. Can be retrieved using the projects table",
				},
				{
					Name:        "team_id",
					Type:        rpc.ColumnTypeString,
					IsParameter: true,
					Description: "The ID of the team. Follow https://vercel.com/docs/accounts/create-a-team#find-your-team-id to find the team ID",
				},
				{
					Name:        "id",
					Type:        rpc.ColumnTypeString,
					Description: "The ID of the deployment",
				},
				{
					Name:        "name",
					Type:        rpc.ColumnTypeString,
					Description: "The name of the deployment",
				},
				{
					Name:        "url",
					Type:        rpc.ColumnTypeString,
					Description: "The URL to see the result of the deployment",
				},
				{
					Name:        "created_at",
					Type:        rpc.ColumnTypeDateTime,
					Description: "The time the deployment was created",
				},
				{
					Name:        "ready_at",
					Type:        rpc.ColumnTypeDateTime,
					Description: "The time the deployment was finished",
				},
				{
					Name:        "building_at",
					Type:        rpc.ColumnTypeDateTime,
					Description: "The time the deployment started building. Can be different from created_at if the deployment was queued",
				},
				{
					Name:        "source",
					Type:        rpc.ColumnTypeString,
					Description: "One of api-trigger-git-deploy | cli | clone/repo | git | import | import/repo | redeploy | v0-web",
				},
				{
					Name:        "state",
					Type:        rpc.ColumnTypeString,
					Description: "One of: BUILDING | ERROR | INITIALIZING | QUEUED | READY | CANCELED | DELETED",
				},
				{
					Name:        "substate",
					Type:        rpc.ColumnTypeString,
					Description: "One of: STAGED | PROMOTED. Promoted means the deployment has seen production traffic",
				},
				{
					Name:        "type",
					Type:        rpc.ColumnTypeString,
					Description: "The deployment type. One of LAMBDA",
				},
				{
					Name:        "target",
					Type:        rpc.ColumnTypeString,
					Description: "One of: production | staging",
				},
				{
					Name:        "creator_email",
					Type:        rpc.ColumnTypeString,
					Description: "The email of the user who initiated the deployment",
				},
				{
					Name:        "creator_name",
					Type:        rpc.ColumnTypeString,
					Description: "The name of the user who initiated the deployment",
				},
				{
					Name:        "inspector_url",
					Type:        rpc.ColumnTypeString,
					Description: "The URL to inspect the deployment (logs, status, etc.)",
				},
				{
					Name:        "github_commit_sha",
					Type:        rpc.ColumnTypeString,
					Description: "The SHA of the commit that triggered the deployment, if triggered by a GitHub commit",
				},
				{
					Name:        "github_commit_author",
					Type:        rpc.ColumnTypeString,
					Description: "The author of the commit that triggered the deployment, if triggered by a GitHub commit",
				},
				{
					Name:        "github_commit_message",
					Type:        rpc.ColumnTypeString,
					Description: "The message of the commit that triggered the deployment, if triggered by a GitHub commit",
				},
			},
		}, nil
}

type deploymentsTable struct {
	token string
	db    *badger.DB
}

type deploymentsCursor struct {
	db    *badger.DB
	token string
	next  int64
}

// Return a slice of rows that will be returned to Anyquery and filtered.
// The second return value is true if the cursor has no more rows to return
//
// The constraints are used for optimization purposes to "pre-filter" the rows
// If the rows returned don't match the constraints, it's not an issue. Anyquery will filter them out
func (t *deploymentsCursor) Query(constraints rpc.QueryConstraint) ([][]interface{}, bool, error) {

	projectID := retrieveArgString(constraints, 0)
	teamID := retrieveArgString(constraints, 1)

	cacheKey := fmt.Sprintf("deployments-%d-%s-%s", t.next, projectID, teamID)

	// Retrieve the deployments
	rows := [][]interface{}{}
	response := &Deployments{}

	// Try to load the cache
	err := t.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte(cacheKey))
		if err != nil {
			return err
		}

		log.Printf("Cache hit: %s", cacheKey)
		return item.Value(func(val []byte) error {
			dec := gob.NewDecoder(bytes.NewReader(val))
			return dec.Decode(response)
		})
	})

	if err != nil {
		log.Printf("Cache miss: %s", cacheKey)
		// Otherwise, fetch the deployments
		endpoint := "https://api.vercel.com/v6/deployments"
		req := client.R().SetHeader("Authorization", "Bearer "+t.token).SetResult(response).
			SetQueryParam("projectId", projectID).
			SetQueryParam("teamId", teamID).
			SetQueryParam("limit", "100")

		if t.next != 0 {
			req.SetQueryParam("until", fmt.Sprintf("%d", t.next))
		}

		res, err := req.Get(endpoint)

		if err != nil {
			return nil, true, fmt.Errorf("failed to fetch deployments: %w", err)
		}

		if res.IsError() {
			return nil, true, fmt.Errorf("failed to fetch deployments(code %s): text %s", res.Status(), res.String())
		}

		// Save the cache
		err = t.db.Update(func(txn *badger.Txn) error {
			buf := bytes.Buffer{}

			enc := gob.NewEncoder(&buf)

			err := enc.Encode(response)
			if err != nil {
				return fmt.Errorf("failed to encode cache: %w", err)
			}

			e := badger.NewEntry([]byte(cacheKey), buf.Bytes()).WithTTL(2 * time.Minute)
			return txn.SetEntry(e)
		})
		if err != nil {
			log.Printf("Failed to save cache: %v", err)
		}
	}

	// Convert the deployments to rows
	for _, deployment := range response.Deployments {
		source := interface{}(nil)
		if deployment.Source != nil {
			source = *deployment.Source
		}
		substate := interface{}(nil)
		if deployment.ReadySubstate != nil {
			substate = *deployment.ReadySubstate
		}
		gitSHA := interface{}(nil)
		gitAuthor := interface{}(nil)
		gitMessage := interface{}(nil)
		if deployment.Meta.GithubCommitSHA != nil {
			gitSHA = *deployment.Meta.GithubCommitSHA
		}
		if deployment.Meta.GithubCommitAuthorLogin != nil {
			gitAuthor = *deployment.Meta.GithubCommitAuthorLogin
		}
		if deployment.Meta.GithubCommitMessage != nil {
			gitMessage = *deployment.Meta.GithubCommitMessage
		}
		target := interface{}(nil)
		if deployment.Target != nil {
			target = *deployment.Target
		}

		rows = append(rows, []interface{}{
			deployment.Uid,
			deployment.Name,
			deployment.URL,
			time.Unix(deployment.CreatedAt, 0).Format(time.RFC3339),
			time.Unix(deployment.Ready, 0).Format(time.RFC3339),
			time.Unix(deployment.BuildingAt, 0).Format(time.RFC3339),
			source,
			deployment.State,
			substate,
			deployment.Type,
			target,
			deployment.Creator.Email,
			deployment.Creator.Username,
			deployment.InspectorURL,
			gitSHA,
			gitAuthor,
			gitMessage,
		})
	}

	// Update the next page
	if response.Pagination.Next == nil {
		t.next = 0
	} else {
		t.next = int64(response.Pagination.Next.(float64))
	}

	return rows, len(rows) < EntriesPerPage || t.next == 0, nil
}

// Create a new cursor that will be used to read rows
func (t *deploymentsTable) CreateReader() rpc.ReaderInterface {
	return &deploymentsCursor{
		db:    t.db,
		token: t.token,
		next:  0,
	}
}

// A destructor to clean up resources
func (t *deploymentsTable) Close() error {
	return nil
}
