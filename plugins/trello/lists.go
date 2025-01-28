package main

import (
	"fmt"

	"github.com/julien040/anyquery/rpc"
)

// A constructor to create a new table instance
// This function is called everytime a new connection is made to the plugin
//
// It should return a new table instance, the database schema and if there is an error
func listsCreator(args rpc.TableCreatorArgs) (rpc.Table, *rpc.DatabaseSchema, error) {
	key := args.UserConfig.GetString("key")
	token := args.UserConfig.GetString("token")

	if key == "" || token == "" {
		return nil, nil, fmt.Errorf("key and token must be set in the plugin configuration to non-empty values")
	}

	return &listsTable{
			key:   key,
			token: token,
		}, &rpc.DatabaseSchema{
			PrimaryKey: 1,
			Columns: []rpc.DatabaseSchemaColumn{
				{
					Name:        "board_id",
					Type:        rpc.ColumnTypeString,
					IsParameter: true,
					IsRequired:  true,
					Description: "The ID of the board. In https://trello.com/b/12345678/board-name, the board ID is 12345678. Can be found in the trello_boards table",
				},
				{
					Name:        "id",
					Type:        rpc.ColumnTypeString,
					Description: "The ID of the list. A list is a collection of cards (often the stages of a project)",
				},
				{
					Name:        "name",
					Type:        rpc.ColumnTypeString,
					Description: "The name of the list",
				},
				{
					Name:        "color",
					Type:        rpc.ColumnTypeString,
					Description: "The color of the list",
				},
				{
					Name:        "subscribed",
					Type:        rpc.ColumnTypeBool,
					Description: "Whether the authenticated user is subscribed to the list",
				},
				{
					Name:        "position",
					Type:        rpc.ColumnTypeFloat,
					Description: "The position of the list in the board",
				},
			},
		}, nil
}

type listsTable struct {
	key   string
	token string
}

type listsCursor struct {
	key   string
	token string
}

// Return a slice of rows that will be returned to Anyquery and filtered.
// The second return value is true if the cursor has no more rows to return
//
// The constraints are used for optimization purposes to "pre-filter" the rows
// If the rows returned don't match the constraints, it's not an issue. Anyquery will filter them out
func (t *listsCursor) Query(constraints rpc.QueryConstraint) ([][]interface{}, bool, error) {
	// Get the board_id from the constraints
	boardID := constraints.GetColumnConstraint(0).GetStringValue()
	if boardID == "" {
		return nil, true, fmt.Errorf("board_id must be set. To do so, use the following query: SELECT * FROM trello_cards('board_id');")
	}

	// Request the rows from the API
	endpoint := "https://api.trello.com/1/boards/{boardID}/lists"
	result := Lists{}

	res, err := client.R().
		SetPathParam("boardID", boardID).
		SetResult(&result).
		SetQueryParam("key", t.key).
		SetQueryParam("token", t.token).
		Get(endpoint)

	if err != nil {
		return nil, true, fmt.Errorf("failed to get lists: %w", err)
	}

	if res.IsError() {
		return nil, true, fmt.Errorf("failed to get lists(%d): %s", res.StatusCode(), res.String())
	}

	// Convert the result to a slice of rows
	rows := make([][]interface{}, 0, len(result))
	for _, list := range result {
		// To ensure color is always a string
		if str, ok := list.Color.(string); ok {
			list.Color = str
		} else {
			list.Color = nil
		}
		rows = append(rows, []interface{}{
			list.ID,
			list.Name,
			list.Color,
			list.Subscribed,
			list.Pos,
		})
	}

	return rows, true, nil

}

// Create a new cursor that will be used to read rows
func (t *listsTable) CreateReader() rpc.ReaderInterface {
	return &listsCursor{
		key:   t.key,
		token: t.token,
	}
}

// A destructor to clean up resources
func (t *listsTable) Close() error {
	return nil
}
