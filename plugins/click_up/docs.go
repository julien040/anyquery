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
func docsCreator(args rpc.TableCreatorArgs) (rpc.Table, *rpc.DatabaseSchema, error) {
	// Example: get a token from the user configuration
	// token := args.UserConfig.GetString("token")
	// if token == "" {
	// 	return nil, nil, fmt.Errorf("token must be set in the plugin configuration")
	// }

	// Example: open a cache connection
	/* cache, err := helper.NewCache(helper.NewCacheArgs{
		Paths:         []string{"docs", "docs" + "_cache"},
		EncryptionKey: []byte("my_secret_key"),
	})*/

	apiKey := args.UserConfig.GetString("api_key")
	if apiKey == "" {
		return nil, nil, fmt.Errorf("api_key must be set in the plugin configuration")
	}

	md5sumApiKey := md5.Sum([]byte(apiKey))
	sha256ApiKey := sha256.Sum256([]byte(apiKey))

	cache, err := helper.NewCache(helper.NewCacheArgs{
		Paths:         []string{"click_up", "docs", fmt.Sprintf("%x", md5sumApiKey)},
		EncryptionKey: []byte(sha256ApiKey[:]),
	})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to open cache: %w", err)
	}

	return &docsTable{
			apiToken: apiKey,
			cache:    cache,
		}, &rpc.DatabaseSchema{
			PrimaryKey: 1,
			Columns: []rpc.DatabaseSchemaColumn{
				{
					Name:        "workspace_id",
					Type:        rpc.ColumnTypeString,
					IsParameter: true,
					IsRequired:  true,
				},
				{
					Name: "doc_id",
					Type: rpc.ColumnTypeString,
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
					Name: "name",
					Type: rpc.ColumnTypeString,
				},
				{
					Name: "parent_id",
					Type: rpc.ColumnTypeString,
				},
				{
					Name: "creator_id",
					Type: rpc.ColumnTypeString,
				},
				{
					Name: "deleted",
					Type: rpc.ColumnTypeBool,
				},
			},
		}, nil
}

// The table struct
// There is one per connection to the plugin and is created by the creator function
// In there, you can store any state you need to read the rows (e.g. a database connection, an API token, etc.)
type docsTable struct {
	apiToken string
	cache    *helper.Cache
}

// The cursor struct
// There is one per query and is created by the createReader function
// In there, you can store any state you need to read the rows (e.g. a database connection from docsTable, an offset, a cursor, etc.)
type docsCursor struct {
	apiToken   string
	cache      *helper.Cache
	nextCursor string
}

// Create a new cursor that will be used to read rows
func (t *docsTable) CreateReader() rpc.ReaderInterface {
	return &docsCursor{
		apiToken: t.apiToken,
		cache:    t.cache,
	}
}

// Return a slice of rows that will be returned to Anyquery and filtered.
// The second return value is true if the cursor has no more rows to return
//
// The constraints are used for optimization purposes to "pre-filter" the rows
// If the rows returned don't match the constraints, it's not an issue. Anyquery will filter them out
func (t *docsCursor) Query(constraints rpc.QueryConstraint) ([][]interface{}, bool, error) {
	workspaceID := constraints.GetColumnConstraint(0).GetStringValue()
	if workspaceID == "" {
		return nil, true, fmt.Errorf("workspace_id must be set")
	}

	cacheKey := fmt.Sprintf("docs_%s_%s", workspaceID, t.nextCursor)

	// Try to get the data from the cache
	// If the data is not in the cache, we'll fetch it from the API
	rows, metadata, err := t.cache.Get(cacheKey)
	if err == nil && len(rows) > 0 {
		t.nextCursor = metadata["next_cursor"].(string)
		return rows, t.nextCursor == "", nil
	}

	// Fetch the data from the API
	body := Docs{}
	resp, err := client.R().
		SetHeader("Authorization", t.apiToken).
		SetQueryParams(map[string]string{}).
		SetResult(&body).
		SetPathParam("workspace_id", workspaceID).
		Get("https://api.clickup.com/api/v3/workspaces/{workspace_id}/docs")

	if err != nil {
		return nil, true, fmt.Errorf("failed to fetch data from the API: %w", err)
	}

	if resp.IsError() {
		return nil, true, fmt.Errorf("failed to fetch data from the API(%d): %s", resp.StatusCode(), resp.String())
	}

	// Compute the rows
	t.nextCursor = body.NextCursor
	rows = make([][]interface{}, 0, len(body.Docs))
	for _, doc := range body.Docs {
		log.Printf("doc: %v", doc)
		rows = append(rows, []interface{}{
			doc.ID,
			convertTime(doc.DateCreated),
			convertTime(doc.DateUpdated),
			doc.Name,
			doc.Parent.ID,
			doc.Creator,
			doc.Deleted,
		})
	}

	// Save the data in the cache
	err = t.cache.Set(cacheKey, rows, map[string]interface{}{
		"next_cursor": t.nextCursor,
	}, time.Minute*5)
	if err != nil {
		return nil, true, fmt.Errorf("failed to save data in the cache: %w", err)
	}

	return rows, t.nextCursor == "", nil
}

// A slice of rows to insert
func (t *docsTable) Insert(rows [][]interface{}) error {
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
func (t *docsTable) Update(rows [][]interface{}) error {
	return nil
}

// A slice of primary keys to delete
func (t *docsTable) Delete(primaryKeys []interface{}) error {
	return nil
}

// A destructor to clean up resources
func (t *docsTable) Close() error {
	return nil
}
