package main

import (
	"crypto/md5"
	"crypto/sha256"
	"fmt"

	"github.com/julien040/anyquery/rpc"
	"github.com/julien040/anyquery/rpc/helper"
)

// A constructor to create a new table instance
// This function is called everytime a new connection is made to the plugin
//
// It should return a new table instance, the database schema and if there is an error
func listsCreator(args rpc.TableCreatorArgs) (rpc.Table, *rpc.DatabaseSchema, error) {
	apiKey := args.UserConfig.GetString("api_key")
	if apiKey == "" {
		return nil, nil, fmt.Errorf("api_key must be set in the plugin configuration")
	}

	md5sumApiKey := md5.Sum([]byte(apiKey))
	sha256ApiKey := sha256.Sum256([]byte(apiKey))

	cache, err := helper.NewCache(helper.NewCacheArgs{
		Paths:         []string{"click_up", "lists", fmt.Sprintf("%x", md5sumApiKey)},
		EncryptionKey: []byte(sha256ApiKey[:]),
	})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to open cache: %w", err)
	}

	return &listsTable{
			apiToken: apiKey,
			cache:    cache,
		}, &rpc.DatabaseSchema{
			HandlesInsert: false,
			HandlesUpdate: false,
			HandlesDelete: false,
			HandleOffset:  false,
			Columns: []rpc.DatabaseSchemaColumn{
				{
					Name:        "folder_id",
					Type:        rpc.ColumnTypeString,
					IsParameter: true,
					IsRequired:  true,
				},
				{
					Name: "list_id",
					Type: rpc.ColumnTypeString,
				},
				{
					Name: "name",
					Type: rpc.ColumnTypeString,
				},
				{
					Name: "order_index",
					Type: rpc.ColumnTypeString,
				},
				{
					Name: "description",
					Type: rpc.ColumnTypeString,
				},
				{
					Name: "task_count",
					Type: rpc.ColumnTypeInt,
				},
				{
					Name: "due_at",
					Type: rpc.ColumnTypeString,
				},
				{
					Name: "start_at",
					Type: rpc.ColumnTypeString,
				},
				{
					Name: "archived",
					Type: rpc.ColumnTypeBool,
				},
			},
		}, nil
}

// The table struct
// There is one per connection to the plugin and is created by the creator function
// In there, you can store any state you need to read the rows (e.g. a database connection, an API token, etc.)
type listsTable struct {
	apiToken string
	cache    *helper.Cache
}

// The cursor struct
// There is one per query and is created by the createReader function
// In there, you can store any state you need to read the rows (e.g. a database connection from listsTable, an offset, a cursor, etc.)
type listsCursor struct {
	apiToken string
	cache    *helper.Cache
}

// Create a new cursor that will be used to read rows
func (t *listsTable) CreateReader() rpc.ReaderInterface {
	return &listsCursor{
		apiToken: t.apiToken,
		cache:    t.cache,
	}
}

// Return a slice of rows that will be returned to Anyquery and filtered.
// The second return value is true if the cursor has no more rows to return
//
// The constraints are used for optimization purposes to "pre-filter" the rows
// If the rows returned don't match the constraints, it's not an issue. Anyquery will filter them out
func (t *listsCursor) Query(constraints rpc.QueryConstraint) ([][]interface{}, bool, error) {
	folderID := constraints.GetColumnConstraint(0).GetStringValue()

	// Try to get the data from the cache
	cacheKey := fmt.Sprintf("lists_%s", folderID)
	rows, _, err := t.cache.Get(cacheKey)
	if err == nil {
		return rows, true, nil
	}

	// Fetch the data from the API
	// If a folder_id is not set, we list the folderless lists
	body := Lists{}
	resp, err := client.R().
		SetHeader("Authorization", t.apiToken).
		SetResult(&body).
		SetPathParam("folder_id", folderID).
		Get("https://api.clickup.com/api/v2/folder/{folder_id}/list")

	if err != nil {
		return nil, true, fmt.Errorf("failed to fetch data from the API: %w", err)
	}

	if resp.IsError() {
		return nil, true, fmt.Errorf("failed to fetch data from the API(%d): %s", resp.StatusCode(), resp.String())
	}

	// Convert the rows
	rows = make([][]interface{}, 0, len(body.Lists))
	for _, list := range body.Lists {
		rows = append(rows, []interface{}{
			list.ID,
			list.Name,
			list.Orderindex,
			list.Content,
			list.TaskCount,
			convertTime(list.DueDate),
			convertTime(list.StartDate),
			list.Archived,
		})
	}

	return rows, true, nil
}

// A slice of rows to insert
func (t *listsTable) Insert(rows [][]interface{}) error {
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
