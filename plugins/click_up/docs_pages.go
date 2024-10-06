package main

import (
	"crypto/md5"
	"crypto/sha256"
	"fmt"
	"time"

	"github.com/julien040/anyquery/rpc"
	"github.com/julien040/anyquery/rpc/helper"
)

// A constructor to create a new table instance
// This function is called everytime a new connection is made to the plugin
//
// It should return a new table instance, the database schema and if there is an error
func docs_pagesCreator(args rpc.TableCreatorArgs) (rpc.Table, *rpc.DatabaseSchema, error) {
	apiKey := args.UserConfig.GetString("api_key")
	if apiKey == "" {
		return nil, nil, fmt.Errorf("api_key must be set in the plugin configuration")
	}

	md5sumApiKey := md5.Sum([]byte(apiKey))
	sha256ApiKey := sha256.Sum256([]byte(apiKey))

	cache, err := helper.NewCache(helper.NewCacheArgs{
		Paths:         []string{"click_up", "pages", fmt.Sprintf("%x", md5sumApiKey)},
		EncryptionKey: []byte(sha256ApiKey[:]),
	})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to open cache: %w", err)
	}

	return &docs_pagesTable{
			apiToken: apiKey,
			cache:    cache,
		}, &rpc.DatabaseSchema{
			Columns: []rpc.DatabaseSchemaColumn{
				{
					Name:        "workspace_id",
					Type:        rpc.ColumnTypeString,
					IsParameter: true,
					IsRequired:  true,
				},
				{
					Name:        "doc_id",
					Type:        rpc.ColumnTypeString,
					IsParameter: true,
					IsRequired:  true,
				},
				{
					Name: "page_id",
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
					Name: "creator_id",
					Type: rpc.ColumnTypeString,
				},
				{
					Name: "content",
					Type: rpc.ColumnTypeString,
				},
				{
					Name: "archived",
					Type: rpc.ColumnTypeBool,
				},
				{
					Name: "deleted",
					Type: rpc.ColumnTypeBool,
				},
				{
					Name: "protected",
					Type: rpc.ColumnTypeBool,
				},
			},
		}, nil
}

// The table struct
// There is one per connection to the plugin and is created by the creator function
// In there, you can store any state you need to read the rows (e.g. a database connection, an API token, etc.)
type docs_pagesTable struct {
	apiToken string
	cache    *helper.Cache
}

// The cursor struct
// There is one per query and is created by the createReader function
// In there, you can store any state you need to read the rows (e.g. a database connection from docs_pagesTable, an offset, a cursor, etc.)
type docs_pagesCursor struct {
	apiToken string
	cache    *helper.Cache
}

// Create a new cursor that will be used to read rows
func (t *docs_pagesTable) CreateReader() rpc.ReaderInterface {
	return &docs_pagesCursor{
		apiToken: t.apiToken,
		cache:    t.cache,
	}
}

// Return a slice of rows that will be returned to Anyquery and filtered.
// The second return value is true if the cursor has no more rows to return
//
// The constraints are used for optimization purposes to "pre-filter" the rows
// If the rows returned don't match the constraints, it's not an issue. Anyquery will filter them out
func (t *docs_pagesCursor) Query(constraints rpc.QueryConstraint) ([][]interface{}, bool, error) {
	workspaceID := constraints.GetColumnConstraint(0).GetStringValue()
	if workspaceID == "" {
		return nil, true, fmt.Errorf("workspace_id must be set")
	}

	docID := constraints.GetColumnConstraint(1).GetStringValue()
	if docID == "" {
		return nil, true, fmt.Errorf("doc_id must be set")
	}

	cacheKey := fmt.Sprintf("pages_%s_%s", workspaceID, docID)

	// Try to fetch the data from the cache
	rows, _, err := t.cache.Get(cacheKey)
	if err == nil && len(rows) > 0 {
		return rows, true, nil
	}

	// Otherwise, fetch the data from the API
	body := Pages{}
	resp, err := client.R().
		SetHeader("Authorization", t.apiToken).
		SetResult(&body).
		SetPathParams(map[string]string{
			"workspace_id": workspaceID,
			"doc_id":       docID,
		}).
		Get("https://api.clickup.com/api/v3/workspaces/{workspace_id}/docs/{doc_id}/pages")

	if err != nil {
		return nil, true, fmt.Errorf("failed to fetch data from the API: %w", err)
	}

	if resp.IsError() {
		return nil, true, fmt.Errorf("failed to fetch data from the API(%d): %s", resp.StatusCode(), resp.String())
	}

	rows = make([][]interface{}, 0, len(body))
	for _, page := range body {
		rows = append(rows, []interface{}{
			page.ID,
			convertTime(page.DateCreated),
			convertTime(page.DateUpdated),
			page.Name,
			page.CreatorID,
			page.Content,
			page.Archived,
			page.Deleted,
			page.Protected,
		})
	}

	// Store the data in the cache
	err = t.cache.Set(cacheKey, rows, nil, time.Minute*5)
	if err != nil {
		return nil, true, fmt.Errorf("failed to save data in the cache: %w", err)
	}

	return rows, true, nil
}

// A slice of rows to insert
func (t *docs_pagesTable) Insert(rows [][]interface{}) error {
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
func (t *docs_pagesTable) Update(rows [][]interface{}) error {
	return nil
}

// A slice of primary keys to delete
func (t *docs_pagesTable) Delete(primaryKeys []interface{}) error {
	return nil
}

// A destructor to clean up resources
func (t *docs_pagesTable) Close() error {
	return nil
}
