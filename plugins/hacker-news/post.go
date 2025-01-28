package main

import (
	"fmt"
	"strconv"
	"time"

	"github.com/julien040/anyquery/rpc"
)

// A constructor to create a new table instance
// This function is called everytime a new connection is made to the plugin
//
// It should return a new table instance, the database schema and if there is an error
func postCreator(args rpc.TableCreatorArgs) (rpc.Table, *rpc.DatabaseSchema, error) {
	return &postTable{}, &rpc.DatabaseSchema{
		HandlesInsert: false,
		HandlesUpdate: false,
		HandlesDelete: false,
		HandleOffset:  false,
		Columns: []rpc.DatabaseSchemaColumn{
			{
				Name:        "id",
				Type:        rpc.ColumnTypeInt,
				IsParameter: true,
				Description: "The ID of the post to find information about",
			},
			{
				Name:        "by",
				Type:        rpc.ColumnTypeString,
				Description: "The username of the author",
			},
			{
				Name:        "created_at",
				Type:        rpc.ColumnTypeDateTime,
				Description: "The creation date of the item",
			},
			{
				Name:        "title",
				Type:        rpc.ColumnTypeString,
				Description: "The title of the item, if any",
			},
			{
				Name:        "url",
				Type:        rpc.ColumnTypeString,
				Description: "The URL to see the item in the browser",
			},
			{
				Name:        "text",
				Type:        rpc.ColumnTypeString,
				Description: "The text of the item, if any",
			},
			{
				Name:        "descendants",
				Type:        rpc.ColumnTypeInt,
				Description: "How many comments the post or comment has",
			},
			{
				Name:        "score",
				Type:        rpc.ColumnTypeInt,
				Description: "The score of the item",
			},
			{
				Name:        "type",
				Type:        rpc.ColumnTypeString,
				Description: "The type of the item. It can be 'job', 'story', 'comment', 'poll', 'pollopt'",
			},
			{
				Name:        "deleted",
				Type:        rpc.ColumnTypeBool,
				Description: "If the item is deleted",
			},
			{
				Name:        "dead",
				Type:        rpc.ColumnTypeBool,
				Description: "If the item is dead",
			},
			{
				Name:        "parent",
				Type:        rpc.ColumnTypeInt,
				Description: "The ID of the parent item",
			},
			{
				Name: "poll",
				Type: rpc.ColumnTypeInt,
			},
			{
				Name:        "kids",
				Type:        rpc.ColumnTypeJSON,
				Description: "A JSON array of the IDs of the kids of the item",
			},
		},
	}, nil
}

type postTable struct {
}

type postCursor struct {
}

// Return a slice of rows that will be returned to Anyquery and filtered.
// The second return value is true if the cursor has no more rows to return
//
// The constraints are used for optimization purposes to "pre-filter" the rows
// If the rows returned don't match the constraints, it's not an issue. Anyquery will filter them out
func (t *postCursor) Query(constraints rpc.QueryConstraint) ([][]interface{}, bool, error) {
	// Get the ID from the constraints
	id := 0
	for _, c := range constraints.Columns {
		if c.ColumnID == 0 {
			switch c.Value.(type) {
			case int:
				id = c.Value.(int)
			case int64:
				id = int(c.Value.(int64))
			case string:
				// Try to parse the string as an int
				var err error
				id, err = strconv.Atoi(c.Value.(string))
				if err != nil {
					return nil, true, fmt.Errorf("invalid id: %s", c.Value.(string))
				}
			}
		}
	}

	if id <= 0 {
		return nil, true, fmt.Errorf("invalid id: %d", id)
	}

	// Fetch the post from the API
	data := HackerNewsAPIResponse{}

	res, err := client.R().SetResult(&data).Get(fmt.Sprintf("https://hacker-news.firebaseio.com/v0/item/%d.json", id))

	// Check for errors
	if err != nil {
		return nil, true, fmt.Errorf("error fetching post: %s", err)
	}

	if res.IsError() {
		return nil, true, fmt.Errorf("error fetching post(%d): %s", res.StatusCode(), res.String())
	}

	if data.ID == 0 {
		return nil, true, fmt.Errorf("post not found")
	}

	// Convert the unix timestamp to rfc3339
	createdAt := time.Unix(int64(data.Time), 0).Format(time.RFC3339)
	rows := [][]interface{}{
		{
			data.By,
			createdAt,
			data.Title,
			data.URL,
			data.Text,
			data.Descendants,
			data.Score,
			data.Type,
			data.Deleted,
			data.Dead,
			data.Parent,
			data.Poll,
			data.Kids,
		},
	}

	return rows, true, nil
}

// Create a new cursor that will be used to read rows
func (t *postTable) CreateReader() rpc.ReaderInterface {
	return &postCursor{}
}

// A destructor to clean up resources
func (t *postTable) Close() error {
	return nil
}
