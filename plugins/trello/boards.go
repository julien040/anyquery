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
					Name: "id",
					Type: rpc.ColumnTypeString,
				},
				{
					Name: "name",
					Type: rpc.ColumnTypeString,
				},
				{
					Name: "description",
					Type: rpc.ColumnTypeString,
				},
				{
					Name: "url",
					Type: rpc.ColumnTypeString,
				},
				{
					Name: "pinned",
					Type: rpc.ColumnTypeBool,
				},
				{
					Name: "starred",
					Type: rpc.ColumnTypeBool,
				},
				{
					Name: "subscribed",
					Type: rpc.ColumnTypeBool,
				},
				{
					Name: "closed_at",
					Type: rpc.ColumnTypeString,
				},
				{
					Name: "last_viewed_at",
					Type: rpc.ColumnTypeString,
				},
				{
					Name: "last_activity_at",
					Type: rpc.ColumnTypeString,
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

// A slice of rows to insert
func (t *boardTable) Insert(rows [][]interface{}) error {
	return nil
}

// A slice of rows to update
// The first element of each row is the primary key
// while the rest are the values to update
// The primary key is therefore present twice
func (t *boardTable) Update(rows [][]interface{}) error {
	return nil
}

// A slice of primary keys to delete
func (t *boardTable) Delete(primaryKeys []interface{}) error {
	return nil
}

// A destructor to clean up resources
func (t *boardTable) Close() error {
	return nil
}
