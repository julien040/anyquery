package main

import (
	"context"
	"fmt"

	"github.com/hashicorp/go-retryablehttp"
	"github.com/julien040/anyquery/rpc"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/option"
	"google.golang.org/api/tasks/v1"
)

// A constructor to create a new table instance
// This function is called everytime a new connection is made to the plugin
//
// It should return a new table instance, the database schema and if there is an error
func listsCreator(args rpc.TableCreatorArgs) (rpc.Table, *rpc.DatabaseSchema, error) {
	var token, clientID, clientSecret string

	if rawInter, ok := args.UserConfig["token"]; ok {
		if token, ok = rawInter.(string); !ok {
			return nil, nil, fmt.Errorf("token should be a string")
		}
		if token == "" {
			return nil, nil, fmt.Errorf("token should not be empty")
		}
	} else {
		return nil, nil, fmt.Errorf("token is required")
	}

	if rawInter, ok := args.UserConfig["client_id"]; ok {
		if clientID, ok = rawInter.(string); !ok {
			return nil, nil, fmt.Errorf("client_id should be a string")
		}
		if clientID == "" {
			return nil, nil, fmt.Errorf("client_id should not be empty")
		}
	} else {
		return nil, nil, fmt.Errorf("client_id is required")
	}

	if rawInter, ok := args.UserConfig["client_secret"]; ok {
		if clientSecret, ok = rawInter.(string); !ok {
			return nil, nil, fmt.Errorf("client_secret should be a string")
		}
		if clientSecret == "" {
			return nil, nil, fmt.Errorf("client_secret should not be empty")
		}
	} else {
		return nil, nil, fmt.Errorf("client_secret is required")
	}

	config := oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		Endpoint:     google.Endpoint,
		Scopes:       []string{tasks.TasksScope},
	}

	oauthClient := config.Client(context.Background(), &oauth2.Token{
		RefreshToken: token,
	})

	retryableClient := retryablehttp.NewClient()
	retryableClient.HTTPClient = oauthClient

	srv, err := tasks.NewService(context.Background(), option.WithHTTPClient(retryableClient.StandardClient()))
	if err != nil {
		return nil, nil, fmt.Errorf("unable to create people service: %w", err)
	}
	return &listsTable{
			srv: srv,
		}, &rpc.DatabaseSchema{
			Columns: []rpc.DatabaseSchemaColumn{
				{
					Name:        "id",
					Type:        rpc.ColumnTypeString,
					Description: "The ID of the list",
				},
				{
					Name:        "title",
					Type:        rpc.ColumnTypeString,
					Description: "The title of the list",
				},
				{
					Name:        "updated_at",
					Type:        rpc.ColumnTypeDateTime,
					Description: "The date the list was last updated",
				},
			},
		}, nil
}

type listsTable struct {
	srv *tasks.Service
}

type listsCursor struct {
	srv *tasks.Service
}

// Return a slice of rows that will be returned to Anyquery and filtered.
// The second return value is true if the cursor has no more rows to return
//
// The constraints are used for optimization purposes to "pre-filter" the rows
// If the rows returned don't match the constraints, it's not an issue. Anyquery will filter them out
func (t *listsCursor) Query(constraints rpc.QueryConstraint) ([][]interface{}, bool, error) {
	// List all the task lists
	lists, err := t.srv.Tasklists.List().Do()
	if err != nil {
		return nil, true, fmt.Errorf("unable to list task lists: %w", err)
	}

	rows := make([][]interface{}, 0, len(lists.Items))
	for _, list := range lists.Items {
		rows = append(rows, []interface{}{
			list.Id,
			list.Title,
			list.Updated,
		})
	}

	return rows, true, nil
}

// Create a new cursor that will be used to read rows
func (t *listsTable) CreateReader() rpc.ReaderInterface {
	return &listsCursor{
		srv: t.srv,
	}
}

// A destructor to clean up resources
func (t *listsTable) Close() error {
	return nil
}
