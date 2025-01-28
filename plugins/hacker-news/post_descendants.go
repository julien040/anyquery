package main

import (
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/julien040/anyquery/rpc"
)

// A constructor to create a new table instance
// This function is called everytime a new connection is made to the plugin
//
// It should return a new table instance, the database schema and if there is an error
func postDescendantCreator(args rpc.TableCreatorArgs) (rpc.Table, *rpc.DatabaseSchema, error) {
	return &postDescendantTable{}, &rpc.DatabaseSchema{
		HandlesInsert: false,
		HandlesUpdate: false,
		HandlesDelete: false,
		HandleOffset:  false,
		Columns: []rpc.DatabaseSchemaColumn{
			{
				Name:        "post_id",
				Type:        rpc.ColumnTypeInt,
				IsParameter: true,
				Description: "The ID of the post",
			},
			{
				Name:        "id",
				Type:        rpc.ColumnTypeInt,
				Description: "The ID of the Hacker News item which is a descendant of post_id",
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
				Name:        "kids",
				Type:        rpc.ColumnTypeJSON,
				Description: "A JSON array of the IDs of the kids of the item",
			},
		},
	}, nil
}

type postDescendantTable struct {
}

type postDescendantCursor struct {
}

// Return a slice of rows that will be returned to Anyquery and filtered.
// The second return value is true if the cursor has no more rows to return
//
// The constraints are used for optimization purposes to "pre-filter" the rows
// If the rows returned don't match the constraints, it's not an issue. Anyquery will filter them out
func (t *postDescendantCursor) Query(constraints rpc.QueryConstraint) ([][]interface{}, bool, error) {
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
	rows := [][]interface{}{}
	mutexRows := sync.Mutex{}

	// Explore the descendants with 16 workers
	buffer := make(chan int, 512) // To avoid blocking the workers
	defer close(buffer)

	// Add the first comments to the buffer
	for _, kid := range data.Kids {
		buffer <- kid
	}

	// Explore the descendants and limit the concurrency to 16
	var wg sync.WaitGroup
	for i := 0; i < 16; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				kid := 0
				var ok bool
				select {
				case kid, ok = <-buffer:
					if !ok {
						return
					}
				default:
					return
				}
				// Fetch the post from the API
				localData := HackerNewsAPIResponse{}

				localRes, err := client.R().SetResult(&localData).Get(fmt.Sprintf("https://hacker-news.firebaseio.com/v0/item/%d.json", kid))
				if err == nil && !localRes.IsError() {
					mutexRows.Lock()
					createdAt := time.Unix(int64(localData.Time), 0).Format(time.RFC3339)
					rows = append(rows, []interface{}{
						localData.ID,
						localData.By,
						createdAt,
						fmt.Sprintf("https://news.ycombinator.com/item?id=%d", localData.ID),
						localData.Text,
						localData.Type,
						localData.Deleted,
						localData.Dead,
						localData.Parent,
						localData.Kids,
					})
					mutexRows.Unlock()
					// Add the kids to the buffer
					for _, kid := range localData.Kids {
						buffer <- kid
					}
				}
			}
		}()
	}

	wg.Wait()

	return rows, true, nil
}

// Create a new cursor that will be used to read rows
func (t *postDescendantTable) CreateReader() rpc.ReaderInterface {
	return &postDescendantCursor{}
}

// A destructor to clean up resources
func (t *postDescendantTable) Close() error {
	return nil
}
