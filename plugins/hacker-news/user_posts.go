package main

import (
	"fmt"
	"sync"
	"time"

	"github.com/julien040/anyquery/rpc"
)

// A constructor to create a new table instance
// This function is called everytime a new connection is made to the plugin
//
// It should return a new table instance, the database schema and if there is an error
func user_postsCreator(args rpc.TableCreatorArgs) (rpc.Table, *rpc.DatabaseSchema, error) {
	return &user_postsTable{}, &rpc.DatabaseSchema{
		HandlesInsert: false,
		HandlesUpdate: false,
		HandlesDelete: false,
		HandleOffset:  false,
		Columns: []rpc.DatabaseSchemaColumn{
			{
				Name:        "id",
				Type:        rpc.ColumnTypeInt,
				IsParameter: true,
				Description: "The ID of the user",
			},
			{
				Name:        "created_at",
				Type:        rpc.ColumnTypeDateTime,
				Description: "The publication date of the post",
			},
			{
				Name:        "title",
				Type:        rpc.ColumnTypeString,
				Description: "The title of the post",
			},
			{
				Name:        "url",
				Type:        rpc.ColumnTypeString,
				Description: "The URL of the post to see it in the browser",
			},
			{
				Name:        "text",
				Type:        rpc.ColumnTypeString,
				Description: "The text of the post",
			},
			{
				Name:        "descendants",
				Type:        rpc.ColumnTypeInt,
				Description: "The number of comments the post has",
			},
			{
				Name:        "score",
				Type:        rpc.ColumnTypeInt,
				Description: "The score of the post",
			},
			{
				Name:        "type",
				Type:        rpc.ColumnTypeString,
				Description: "The type of the post",
			},
			{
				Name: "deleted",
				Type: rpc.ColumnTypeBool,
			},
			{
				Name: "dead",
				Type: rpc.ColumnTypeBool,
			},
			{
				Name:        "parent",
				Type:        rpc.ColumnTypeInt,
				Description: "The ID of the parent post, if any",
			},
			{
				Name: "poll",
				Type: rpc.ColumnTypeInt,
			},
			{
				Name:        "kids",
				Type:        rpc.ColumnTypeString,
				Description: "A JSON array of the IDs of the comments",
			},
		},
	}, nil
}

type user_postsTable struct {
}

type user_postsCursor struct {
}

// Return a slice of rows that will be returned to Anyquery and filtered.
// The second return value is true if the cursor has no more rows to return
//
// The constraints are used for optimization purposes to "pre-filter" the rows
// If the rows returned don't match the constraints, it's not an issue. Anyquery will filter them out
func (t *user_postsCursor) Query(constraints rpc.QueryConstraint) ([][]interface{}, bool, error) {
	limit := 2 << 31
	if constraints.Limit > 0 {
		limit = constraints.Limit
	}
	userID := ""
	for _, constraint := range constraints.Columns {
		if constraint.ColumnID == 0 {
			if parsedStr, ok := constraint.Value.(string); ok {
				userID = parsedStr
			}
		}
	}

	if userID == "" {
		return nil, true, fmt.Errorf("user_id is a string and is required")
	}

	// Call the API
	data := HackerNewsUserAPIResponse{}
	res, err := client.R().SetResult(&data).Get(fmt.Sprintf("https://hacker-news.firebaseio.com/v0/user/%s.json", userID))

	if err != nil {
		return nil, true, fmt.Errorf("error fetching user: %s", err)
	}

	if res.IsError() {
		return nil, true, fmt.Errorf("error fetching user(code %d): %s", res.StatusCode(), res.String())
	}

	// Fetch the posts with 32 concurrent requests
	rows := [][]interface{}{}
	buffer := make(chan int, 256)
	wg := sync.WaitGroup{}
	for i := 0; i < 32; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				id, ok := <-buffer
				if !ok {
					return
				}
				localData := HackerNewsAPIResponse{}
				res, err := client.R().SetResult(&localData).Get(fmt.Sprintf("https://hacker-news.firebaseio.com/v0/item/%d.json", id))
				if err != nil {
					return
				}

				if res.IsError() {
					return
				}

				created_at := time.Unix(int64(localData.Time), 0).Format(time.RFC3339)
				if localData.Type == "comment" {
					localData.URL = fmt.Sprintf("https://news.ycombinator.com/item?id=%d", localData.ID)
				}
				rows = append(rows, []interface{}{
					created_at,
					localData.Title,
					localData.URL,
					localData.Text,
					localData.Descendants,
					localData.Score,
					localData.Type,
					localData.Deleted,
					localData.Dead,
					localData.Parent,
					localData.Poll,
					localData.Kids,
				})
			}

		}()
	}

	for i, postID := range data.Submitted {
		if i >= limit {
			break
		}

		buffer <- postID
	}

	close(buffer)

	wg.Wait()

	return rows, true, nil
}

// Create a new cursor that will be used to read rows
func (t *user_postsTable) CreateReader() rpc.ReaderInterface {
	return &user_postsCursor{}
}

// A destructor to clean up resources
func (t *user_postsTable) Close() error {
	return nil
}
