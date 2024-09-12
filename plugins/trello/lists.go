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
				},
				{
					Name: "id",
					Type: rpc.ColumnTypeString,
				},
				{
					Name: "name",
					Type: rpc.ColumnTypeString,
				},
				{
					Name: "color",
					Type: rpc.ColumnTypeString,
				},
				{
					Name: "subscribed",
					Type: rpc.ColumnTypeBool,
				},
				{
					Name: "position",
					Type: rpc.ColumnTypeFloat,
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

// A slice of rows to insert
func (t *listsTable) Insert(rows [][]interface{}) error {
	return nil
}

// A slice of rows to update
// The first element of each row is the primary key
// while the rest are the values to update
// The primary key is therefore present twice
func (t *listsTable) Update(rows [][]interface{}) error {
	return nil
}

// A slice of primary keys to delete
func (t *listsTable) Delete(primaryKeys []interface{}) error {
	return nil
}

// A destructor to clean up resources
func (t *listsTable) Close() error {
	return nil
}
