package main

import (
	"crypto/md5"
	"crypto/sha256"
	"fmt"
	"log"
	"time"

	"github.com/julien040/anyquery/rpc"
	"github.com/julien040/anyquery/rpc/helper"
)

// A constructor to create a new table instance
// This function is called everytime a new connection is made to the plugin
//
// It should return a new table instance, the database schema and if there is an error
func tasksCreator(args rpc.TableCreatorArgs) (rpc.Table, *rpc.DatabaseSchema, error) {
	// Example: get a token from the user configuration
	// token := args.UserConfig.GetString("token")
	// if token == "" {
	// 	return nil, nil, fmt.Errorf("token must be set in the plugin configuration")
	// }

	// Example: open a cache connection
	/* cache, err := helper.NewCache(helper.NewCacheArgs{
		Paths:         []string{"tasks", "tasks" + "_cache"},
		EncryptionKey: []byte("my_secret_key"),
	})*/

	apiKey := args.UserConfig.GetString("api_key")
	if apiKey == "" {
		return nil, nil, fmt.Errorf("api_key must be set in the plugin configuration")
	}

	md5sumApiKey := md5.Sum([]byte(apiKey))
	sha256ApiKey := sha256.Sum256([]byte(apiKey))

	cache, err := helper.NewCache(helper.NewCacheArgs{
		Paths:         []string{"click_up", "tasks", fmt.Sprintf("%x", md5sumApiKey)},
		EncryptionKey: []byte(sha256ApiKey[:]),
	})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to open cache: %w", err)
	}

	return &tasksTable{
			apiToken: apiKey,
			cache:    cache,
		}, &rpc.DatabaseSchema{
			PrimaryKey: 1,
			Columns: []rpc.DatabaseSchemaColumn{
				{
					Name:        "list_id",
					Type:        rpc.ColumnTypeString,
					IsParameter: true,
					IsRequired:  true,
				},
				{
					Name: "task_id",
					Type: rpc.ColumnTypeString,
				},
				{
					Name: "description",
					Type: rpc.ColumnTypeString,
				},
				{
					Name: "status",
					Type: rpc.ColumnTypeString,
				},
				{
					Name: "order_index",
					Type: rpc.ColumnTypeInt,
				},
				{
					Name: "created_at",
					Type: rpc.ColumnTypeString,
				},
				{
					Name: "updated_at",
					Type: rpc.ColumnTypeString,
				},
				{
					Name: "closed_at",
					Type: rpc.ColumnTypeString,
				},
				{
					Name: "done_at",
					Type: rpc.ColumnTypeString,
				},
				{
					Name: "created_by",
					Type: rpc.ColumnTypeString,
				},
				{
					Name: "started_at",
					Type: rpc.ColumnTypeString,
				},
				{
					Name: "due_at",
					Type: rpc.ColumnTypeString,
				},
				{
					Name: "estimated_time",
					Type: rpc.ColumnTypeInt,
				},
				{
					Name: "time_spent",
					Type: rpc.ColumnTypeInt,
				},
				{
					Name: "assignees",
					Type: rpc.ColumnTypeString,
				},
				{
					Name: "watchers",
					Type: rpc.ColumnTypeString,
				},
				{
					Name: "tags",
					Type: rpc.ColumnTypeString,
				},
				{
					Name: "custom_fields",
					Type: rpc.ColumnTypeString,
				},
				{
					Name: "parent",
					Type: rpc.ColumnTypeString,
				},
				{
					Name: "project_id",
					Type: rpc.ColumnTypeString,
				},
				{
					Name: "folder_id",
					Type: rpc.ColumnTypeString,
				},
				{
					Name: "space_id",
					Type: rpc.ColumnTypeString,
				},
				{
					Name: "url",
					Type: rpc.ColumnTypeString,
				},
			},
		}, nil
}

// The table struct
// There is one per connection to the plugin and is created by the creator function
// In there, you can store any state you need to read the rows (e.g. a database connection, an API token, etc.)
type tasksTable struct {
	apiToken string
	cache    *helper.Cache
}

// The cursor struct
// There is one per query and is created by the createReader function
// In there, you can store any state you need to read the rows (e.g. a database connection from tasksTable, an offset, a cursor, etc.)
type tasksCursor struct {
	apiToken string
	cache    *helper.Cache
	pageID   int
}

// Create a new cursor that will be used to read rows
func (t *tasksTable) CreateReader() rpc.ReaderInterface {
	return &tasksCursor{
		apiToken: t.apiToken,
		cache:    t.cache,
	}
}

// Return a slice of rows that will be returned to Anyquery and filtered.
// The second return value is true if the cursor has no more rows to return
//
// The constraints are used for optimization purposes to "pre-filter" the rows
// If the rows returned don't match the constraints, it's not an issue. Anyquery will filter them out
func (t *tasksCursor) Query(constraints rpc.QueryConstraint) ([][]interface{}, bool, error) {
	listID := constraints.GetColumnConstraint(0).GetStringValue()
	if listID == "" {
		return nil, true, fmt.Errorf("list_id must be set")
	}

	// We'll try to get the data from the cache
	// If it fails, we'll fetch it from the API
	cacheKey := fmt.Sprintf("tasks_%s_%d", listID, t.pageID)
	var rows [][]interface{}
	var err error
	rows, _, err = t.cache.Get(cacheKey)
	if err == nil && len(rows) > 0 {
		t.pageID++
		return rows, len(rows) < 100, nil
	}

	// Fetch the data from the API
	body := Tasks{}
	resp, err := client.R().
		SetHeader("Authorization", t.apiToken).
		SetResult(&body).
		SetPathParam("list_id", listID).
		SetQueryParams(map[string]string{
			"page":                         fmt.Sprintf("%d", t.pageID),
			"limit":                        "100",
			"include_closed":               "true",
			"include_markdown_description": "true",
		}).
		Get("https://api.clickup.com/api/v2/list/{list_id}/task")

	if err != nil {
		return nil, true, fmt.Errorf("failed to fetch tasks: %w", err)
	}

	if resp.IsError() {
		return nil, true, fmt.Errorf("failed to fetch tasks(%d): %s", resp.StatusCode(), resp.String())
	}

	// Compute the rows
	rows = make([][]interface{}, 0, len(body.Tasks))
	for _, task := range body.Tasks {
		tags := []string{}
		for _, tag := range task.Tags {
			tags = append(tags, tag.Name)
		}
		customField := map[string]interface{}{}
		for _, field := range task.CustomFields {
			customField[field.Name] = field.Value
		}
		row := []interface{}{
			task.ID,
			helper.Serialize(task.Description),
			task.Status.Status,
			task.Orderindex,
			convertTime(task.DateCreated),
			convertTime(task.DateUpdated),
			convertTime(task.DateClosed),
			convertTime(task.DateDone),
			helper.Serialize(task.Creator),
			convertTime(task.StartDate),
			convertTime(task.DueDate),
			helper.Serialize(task.TimeEstimate),
			helper.Serialize(task.TimeSpent),
			helper.Serialize(task.Assignees),
			helper.Serialize(task.Watchers),
			tags,
			helper.Serialize(customField),
			helper.Serialize(task.Parent),
			task.Project.ID,
			task.Folder.ID,
			task.Space.ID,
			task.URL,
		}

		rows = append(rows, row)
	}

	// Save the data in the cache
	err = t.cache.Set(cacheKey, rows, nil, time.Minute*5)
	if err != nil {
		log.Printf("failed to save cache: %v", err)
	}

	return rows, len(rows) < 100, nil
}

// A slice of rows to insert
func (t *tasksTable) Insert(rows [][]interface{}) error {
	// Example: insert the rows in a database
	// for _, row := range rows {
	// 	err := db.Insert(row[0], row[1], row[2])
	// 	if err != nil {
	// 		return err
	// 	}
	return nil
}

// A slice of rows to update
// The first element of each row is the primary key
// while the rest are the values to update
// The primary key is therefore present twice
func (t *tasksTable) Update(rows [][]interface{}) error {
	return nil
}

// A slice of primary keys to delete
func (t *tasksTable) Delete(primaryKeys []interface{}) error {
	return nil
}

// A destructor to clean up resources
func (t *tasksTable) Close() error {
	return nil
}
