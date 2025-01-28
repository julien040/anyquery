package main

import (
	"fmt"
	"time"

	"github.com/julien040/anyquery/rpc"
)

// A constructor to create a new table instance
// This function is called everytime a new connection is made to the plugin
//
// It should return a new table instance, the database schema and if there is an error
func user_dataCreator(args rpc.TableCreatorArgs) (rpc.Table, *rpc.DatabaseSchema, error) {
	return &user_dataTable{}, &rpc.DatabaseSchema{
		Columns: []rpc.DatabaseSchemaColumn{
			{
				Name:        "id",
				Type:        rpc.ColumnTypeString,
				IsParameter: true,
				Description: "The ID of the user",
			},
			{
				Name:        "created_at",
				Type:        rpc.ColumnTypeDateTime,
				Description: "The creation date of the account",
			},
			{
				Name:        "karma",
				Type:        rpc.ColumnTypeInt,
				Description: "The karma of the user",
			},
			{
				Name:        "about",
				Type:        rpc.ColumnTypeString,
				Description: "A small bio of the user",
			},
			{
				Name:        "post_id",
				Type:        rpc.ColumnTypeInt,
				Description: "The ID of one of the user's submitted posts",
			},
		},
	}, nil
}

type user_dataTable struct {
}

type user_dataCursor struct {
}

type userDataResponse struct {
	CreatedAt int64   `json:"created"`
	Karma     int64   `json:"karma"`
	About     string  `json:"about"`
	Id        string  `json:"id"`
	Submitted []int64 `json:"submitted"`
}

// Return a slice of rows that will be returned to Anyquery and filtered.
// The second return value is true if the cursor has no more rows to return
//
// The constraints are used for optimization purposes to "pre-filter" the rows
// If the rows returned don't match the constraints, it's not an issue. Anyquery will filter them out
func (t *user_dataCursor) Query(constraints rpc.QueryConstraint) ([][]interface{}, bool, error) {
	user := ""
	for _, c := range constraints.Columns {
		if c.ColumnID == 0 {
			var ok bool
			user, ok = c.Value.(string)
			if !ok {
				return nil, true, fmt.Errorf("id is a string and is required")
			}
		}
	}
	if user == "" {
		return nil, true, fmt.Errorf("you must pass a user id in the table arguments")
	}

	// Get the user data
	data := userDataResponse{}

	res, err := client.R().SetResult(&data).Get(fmt.Sprintf("https://hacker-news.firebaseio.com/v0/user/%s.json", user))
	if err != nil {
		return nil, true, fmt.Errorf("failed to fetch user data: %w", err)
	}

	if res.IsError() {
		return nil, true, fmt.Errorf("failed to fetch user data(%d): %s", res.StatusCode(), res.String())
	}

	// Create the row
	rows := make([][]interface{}, 0, len(data.Submitted))

	for _, id := range data.Submitted {
		rows = append(rows, []interface{}{
			time.Unix(data.CreatedAt, 0).Format(time.RFC3339),
			data.Karma,
			data.About,
			id,
		})
	}

	return rows, true, nil

}

// Create a new cursor that will be used to read rows
func (t *user_dataTable) CreateReader() rpc.ReaderInterface {
	return &user_dataCursor{}
}

// A destructor to clean up resources
func (t *user_dataTable) Close() error {
	return nil
}
