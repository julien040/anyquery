package main

import (
	"fmt"

	"github.com/julien040/anyquery/rpc"
)

// A constructor to create a new table instance
// This function is called everytime a new connection is made to the plugin
//
// It should return a new table instance, the database schema and if there is an error
func boardsCreator(args rpc.TableCreatorArgs) (rpc.Table, *rpc.DatabaseSchema, error) {
	key := args.UserConfig.GetString("key")
	token := args.UserConfig.GetString("token")
	userID := args.UserConfig.GetString("user_id")

	if key == "" || token == "" || userID == "" {
		return nil, nil, fmt.Errorf("key, token and user_id must be set in the plugin configuration to non-empty values")
	}

	return &boardTable{
			userID: userID,
			key:    key,
			token:  token,
		}, &rpc.DatabaseSchema{
			Columns: []rpc.DatabaseSchemaColumn{
				{
					Name:        "id",
					Type:        rpc.ColumnTypeString,
					Description: "The ID of the board. In https://trello.com/b/12345678/board-name, the board ID is 12345678",
				},
				{
					Name:        "name",
					Type:        rpc.ColumnTypeString,
					Description: "The name of the board",
				},
				{
					Name:        "description",
					Type:        rpc.ColumnTypeString,
					Description: "The description of the board",
				},
				{
					Name:        "url",
					Type:        rpc.ColumnTypeString,
					Description: "The URL of the board to view it in the browser",
				},
				{
					Name:        "pinned",
					Type:        rpc.ColumnTypeBool,
					Description: "Whether the board is pinned",
				},
				{
					Name:        "starred",
					Type:        rpc.ColumnTypeBool,
					Description: "Whether the board is starred",
				},
				{
					Name:        "subscribed",
					Type:        rpc.ColumnTypeBool,
					Description: "Whether the authenticated user is subscribed to the board",
				},
				{
					Name:        "closed_at",
					Type:        rpc.ColumnTypeDateTime,
					Description: "When the board was closed",
				},
				{
					Name:        "last_viewed_at",
					Type:        rpc.ColumnTypeDateTime,
					Description: "When the board was last viewed",
				},
				{
					Name:        "last_activity_at",
					Type:        rpc.ColumnTypeDateTime,
					Description: "When the board was last modified",
				},
			},
		}, nil
}

type boardTable struct {
	userID string
	key    string
	token  string
}

type boardCursor struct {
	userID string
	key    string
	token  string
}

// Return a slice of rows that will be returned to Anyquery and filtered.
// The second return value is true if the cursor has no more rows to return
//
// The constraints are used for optimization purposes to "pre-filter" the rows
// If the rows returned don't match the constraints, it's not an issue. Anyquery will filter them out
func (t *boardCursor) Query(constraints rpc.QueryConstraint) ([][]interface{}, bool, error) {
	endpoint := "https://api.trello.com/1/members/{userID}/boards"

	// Get the boards from the API
	body := Boards{}
	res, err := client.R().SetPathParam("userID", t.userID).
		SetQueryParam("key", t.key).
		SetQueryParam("token", t.token).
		SetHeader("Accept", "application/json").
		SetResult(&body).
		Get(endpoint)

	if err != nil {
		return nil, false, fmt.Errorf("failed to get the boards: %w", err)
	}

	if res.IsError() {
		return nil, false, fmt.Errorf("failed to get the boards(%d): %s", res.StatusCode(), res.String())
	}

	// Convert the boards to rows
	rows := make([][]interface{}, 0, len(body))
	for _, board := range body {
		closedAt := interface{}(nil)
		if board.DateClosed != nil {
			// Check if closedAt is a string
			if _, ok := board.DateClosed.(string); ok {
				closedAt = board.DateClosed.(string)
			}
		}
		lastActivityAt := interface{}(nil)
		if board.DateLastActivity != nil {
			lastActivityAt = *board.DateLastActivity
		}
		rows = append(rows, []interface{}{
			board.ID,
			board.Name,
			board.Desc,
			board.URL,
			board.Pinned,
			board.Starred,
			board.Subscribed,
			closedAt,
			board.DateLastView,
			lastActivityAt,
		})
	}

	return rows, true, nil

}

// Create a new cursor that will be used to read rows
func (t *boardTable) CreateReader() rpc.ReaderInterface {
	return &boardCursor{
		userID: t.userID,
		key:    t.key,
		token:  t.token,
	}
}

// A destructor to clean up resources
func (t *boardTable) Close() error {
	return nil
}
