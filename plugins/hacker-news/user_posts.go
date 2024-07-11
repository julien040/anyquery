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
			},
			{
				Name: "created_at",
				Type: rpc.ColumnTypeString,
			},
			{
				Name: "title",
				Type: rpc.ColumnTypeString,
			},
			{
				Name: "url",
				Type: rpc.ColumnTypeString,
			},
			{
				Name: "text",
				Type: rpc.ColumnTypeString,
			},
			{
				Name: "descendants",
				Type: rpc.ColumnTypeInt,
			},
			{
				Name: "score",
				Type: rpc.ColumnTypeInt,
			},
			{
				Name: "type",
				Type: rpc.ColumnTypeString,
			},
			{
				Name: "deleted",
				Type: rpc.ColumnTypeInt,
			},
			{
				Name: "dead",
				Type: rpc.ColumnTypeInt,
			},
			{
				Name: "parent",
				Type: rpc.ColumnTypeInt,
			},
			{
				Name: "poll",
				Type: rpc.ColumnTypeInt,
			},
			{
				Name: "kids",
				Type: rpc.ColumnTypeString,
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

// A slice of rows to insert
func (t *user_postsTable) Insert(rows [][]interface{}) error {
	return nil
}

// A slice of rows to update
// The first element of each row is the primary key
// while the rest are the values to update
// The primary key is therefore present twice
func (t *user_postsTable) Update(rows [][]interface{}) error {
	return nil
}

// A slice of primary keys to delete
func (t *user_postsTable) Delete(primaryKeys []interface{}) error {
	return nil
}

// A destructor to clean up resources
func (t *user_postsTable) Close() error {
	return nil
}
