package main

import (
	"fmt"

	"github.com/julien040/anyquery/rpc"
)

// A constructor to create a new table instance
// This function is called everytime a new connection is made to the plugin
//
// It should return a new table instance, the database schema and if there is an error
func whoamiCreator(args rpc.TableCreatorArgs) (rpc.Table, *rpc.DatabaseSchema, error) {
	apiKey := args.UserConfig.GetString("api_key")
	if apiKey == "" {
		return nil, nil, fmt.Errorf("api_key must be set in the plugin configuration")
	}

	return &whoamiTable{
			token: apiKey,
		}, &rpc.DatabaseSchema{
			Columns: []rpc.DatabaseSchemaColumn{
				{
					Name:        "id",
					Type:        rpc.ColumnTypeInt,
					Description: "The ID of the authenticated user",
				},
				{
					Name:        "username",
					Type:        rpc.ColumnTypeString,
					Description: "The username of the authenticated user",
				},
				{
					Name:        "email",
					Type:        rpc.ColumnTypeString,
					Description: "The email of the authenticated user",
				},
				{
					Name:        "color",
					Type:        rpc.ColumnTypeString,
					Description: "The color of the authenticated user",
				},
				{
					Name:        "profile_picture",
					Type:        rpc.ColumnTypeString,
					Description: "URL of the profile picture of the authenticated user",
				},
				{
					Name:        "initials",
					Type:        rpc.ColumnTypeString,
					Description: "The initials of the authenticated user",
				},
				{
					Name:        "week_start_day",
					Type:        rpc.ColumnTypeInt,
					Description: "The day of the week the user starts on",
				},
				{
					Name:        "timezone",
					Type:        rpc.ColumnTypeString,
					Description: "The timezone of the authenticated user",
				},
			},
		}, nil
}

// The table struct
// There is one per connection to the plugin and is created by the creator function
// In there, you can store any state you need to read the rows (e.g. a database connection, an API token, etc.)
type whoamiTable struct {
	token string
}

// The cursor struct
// There is one per query and is created by the createReader function
// In there, you can store any state you need to read the rows (e.g. a database connection from whoamiTable, an offset, a cursor, etc.)
type whoamiCursor struct {
	token string
}

// Create a new cursor that will be used to read rows
func (t *whoamiTable) CreateReader() rpc.ReaderInterface {
	return &whoamiCursor{
		token: t.token,
	}
}

// Return a slice of rows that will be returned to Anyquery and filtered.
// The second return value is true if the cursor has no more rows to return
//
// The constraints are used for optimization purposes to "pre-filter" the rows
// If the rows returned don't match the constraints, it's not an issue. Anyquery will filter them out
func (t *whoamiCursor) Query(constraints rpc.QueryConstraint) ([][]interface{}, bool, error) {

	body := User{}
	resp, err := client.R().
		SetHeader("Authorization", t.token).
		SetResult(&body).
		Get("https://api.clickup.com/api/v2/user")

	if err != nil {
		return nil, true, fmt.Errorf("error while fetching data: %v", err)
	}

	if resp.IsError() {
		return nil, true, fmt.Errorf("error while fetching data(%d): %s", resp.StatusCode(), resp.String())
	}

	return [][]interface{}{
		{
			body.User.ID,
			body.User.Username,
			body.User.Email,
			body.User.Color,
			body.User.ProfilePicture,
			body.User.Initials,
			body.User.WeekStartDay,
			body.User.Timezone,
		},
	}, true, nil
}

// A destructor to clean up resources
func (t *whoamiTable) Close() error {
	return nil
}
