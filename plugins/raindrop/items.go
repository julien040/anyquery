package main

import (
	"bytes"
	"crypto/md5"
	"encoding/gob"
	"encoding/json"
	"fmt"
	"log"
	"path"
	"strconv"
	"time"

	"github.com/adrg/xdg"
	"github.com/dgraph-io/badger/v4"
	"github.com/julien040/anyquery/rpc"
)

// A constructor to create a new table instance
// This function is called everytime a new connection is made to the plugin
//
// It should return a new table instance, the database schema and if there is an error
func raindropCreator(args rpc.TableCreatorArgs) (rpc.Table, *rpc.DatabaseSchema, error) {
	// Get the token from the arguments
	rawToken, ok := args.UserConfig["token"]
	if !ok {
		return nil, nil, fmt.Errorf("missing token in user config")
	}

	token, ok := rawToken.(string)
	if !ok {
		return nil, nil, fmt.Errorf("token must be a string")
	}

	if token == "" {
		return nil, nil, fmt.Errorf("token cannot be empty")
	}

	// Open the database
	md5token := md5.Sum([]byte(token))

	dbPath := path.Join(xdg.CacheHome, "anyquery", "plugins", "raindrop", fmt.Sprintf("%x", md5token))

	option := badger.DefaultOptions(dbPath).WithEncryptionKey(md5token[:]).
		WithNumVersionsToKeep(1).WithCompactL0OnClose(true).WithValueLogFileSize(2 << 22).
		WithIndexCacheSize(2 << 24)

	db, err := badger.Open(option)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to open database: %w", err)
	}

	return &raindropTable{
			token: token,
			db:    db,
		}, &rpc.DatabaseSchema{
			HandlesInsert: true,
			HandlesUpdate: false,
			HandlesDelete: true,
			HandleOffset:  false,
			BufferInsert:  99,
			BufferUpdate:  0,
			BufferDelete:  99,
			PrimaryKey:    0,
			Columns: []rpc.DatabaseSchemaColumn{
				{
					Name:        "id",
					Type:        rpc.ColumnTypeInt,
					Description: "The ID of the drop (the Raindrop bookmark)",
				},
				{
					Name:        "link",
					Type:        rpc.ColumnTypeString,
					Description: "The URL to see the bookmark",
				},
				{
					Name:        "title",
					Type:        rpc.ColumnTypeString,
					Description: "The title of the bookmark",
				},
				{
					Name:        "excerpt",
					Type:        rpc.ColumnTypeString,
					Description: "A small summary of the bookmark",
				},
				{
					Name:        "note",
					Type:        rpc.ColumnTypeString,
					Description: "Markdown notes of the bookmark written by the user",
				},
				{
					Name:        "user_id",
					Type:        rpc.ColumnTypeString,
					Description: "The ID of the user who created the bookmark",
				},
				{
					Name:        "cover",
					Type:        rpc.ColumnTypeString,
					Description: "The URL of the cover image of the bookmark",
				},
				{
					Name:        "tags",
					Type:        rpc.ColumnTypeJSON,
					Description: "A JSON array of tags",
				},
				{
					Name:        "important",
					Type:        rpc.ColumnTypeBool,
					Description: "Whether the bookmark is marked as important",
				},
				{
					Name:        "removed",
					Type:        rpc.ColumnTypeBool,
					Description: "Whether the bookmark is removed",
				},
				{
					Name:        "created_at",
					Type:        rpc.ColumnTypeDateTime,
					Description: "The date the bookmark was created (RFC3339)",
				},
				{
					Name:        "last_updated_at",
					Type:        rpc.ColumnTypeDateTime,
					Description: "The date the bookmark was last updated (RFC3339)",
				},
				{
					Name:        "domain",
					Type:        rpc.ColumnTypeString,
					Description: "The hostname of the URL of the bookmark",
				},
				{
					Name:        "collection_id",
					Type:        rpc.ColumnTypeInt,
					Description: "The ID of the collection the bookmark is in. -1 if the bookmark is not in a collection. -99 if deleted, and 0 if in the root collection. You can find the collection_id in the URL. For example, in https://app.raindrop.io/my/27189677, the collection_id is 27189677.",
				},
				{
					Name:        "reminder",
					Type:        rpc.ColumnTypeDateTime,
					Description: "The date the Raindrop app will send a reminder to the user to read the bookmark. If the bookmark is not a reminder, this field is null.",
				},
			},
		}, nil
}

type raindropTable struct {
	token string
	db    *badger.DB
}

type raindropCursor struct {
	token   string
	db      *badger.DB
	page    int
	perPage int
}

// Return a slice of rows that will be returned to Anyquery and filtered.
// The second return value is true if the cursor has no more rows to return
//
// The constraints are used for optimization purposes to "pre-filter" the rows
// If the rows returned don't match the constraints, it's not an issue. Anyquery will filter them out
func (t *raindropCursor) Query(constraints rpc.QueryConstraint) ([][]interface{}, bool, error) {
	// Try to get the data from the cache
	cacheKey := fmt.Sprintf("raindrop-%d-%d", t.page, t.perPage)
	rows := [][]interface{}{}

	err := t.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte(cacheKey))
		if err != nil {
			return err
		}
		// Unmarshal with gob
		return item.Value(func(val []byte) error {
			decoded := gob.NewDecoder(bytes.NewReader(val))
			return decoded.Decode(&rows)
		})
	})
	if err == nil {
		t.page++
		return rows, len(rows) < t.perPage, nil
	}

	apiData := &RaindropListItemResponse{}

	// Get the data from the API
	res, err := client.R().
		SetHeader("Authorization", fmt.Sprintf("Bearer %s", t.token)).
		SetQueryParams(map[string]string{
			"perpage": fmt.Sprintf("%d", t.perPage),
			"page":    fmt.Sprintf("%d", t.page),
			"sort":    "-created",
		}).
		SetResult(apiData).
		Get("https://api.raindrop.io/rest/v1/raindrops/0")

	if err != nil {
		return nil, true, fmt.Errorf("failed to get data from the API: %w", err)
	}

	if res.IsError() {
		return nil, true, fmt.Errorf("failed to get data from the API: %s", res.String())
	}

	// Compute the rows
	for _, item := range apiData.Items {
		important := false
		if item.Important != nil {
			important = *item.Important
		}

		reminder := interface{}(nil)
		if item.Reminder != nil {
			reminder = item.Reminder.Date
		}

		rows = append(rows, []interface{}{
			item.ID,
			item.Link,
			item.Title,
			item.Excerpt,
			item.Note,
			item.User.ID,
			item.Cover,
			item.Tags,
			important,
			item.Removed,
			item.Created,
			item.LastUpdate,
			item.Domain,
			item.Collection.ID,
			reminder,
		})
	}

	err = t.db.Update(func(txn *badger.Txn) error {
		// Marshal with gob
		var buf bytes.Buffer
		enc := gob.NewEncoder(&buf)
		if err := enc.Encode(rows); err != nil {
			return err
		}
		e := badger.NewEntry([]byte(cacheKey), buf.Bytes()).WithTTL(time.Hour)
		return txn.SetEntry(e)
	})
	if err != nil {
		log.Printf("failed to cache data: %v", err)
	}

	t.page++

	return rows, len(rows) < t.perPage, nil
}

// Create a new cursor that will be used to read rows
func (t *raindropTable) CreateReader() rpc.ReaderInterface {
	return &raindropCursor{
		token:   t.token,
		db:      t.db,
		page:    0,
		perPage: 50,
	}
}

// A slice of rows to insert
func (t *raindropTable) Insert(rows [][]interface{}) error {
	request := &MultipleCreateItemRequest{
		Items: []CreateItem{},
	}

	for i, row := range rows {
		var ok bool
		var rawStr string
		item := CreateItem{}
		// Add the properties
		// Title
		rawStr, ok = row[1].(string)
		if ok {
			item.Link = rawStr
		}

		// Title
		rawStr, ok = row[2].(string)
		if ok {
			item.Title = rawStr
		}

		// Excerpt
		rawStr, ok = row[3].(string)
		if ok {
			item.Excerpt = rawStr
		}

		// Important
		switch rawVal := row[8].(type) {
		case int64:
			item.Important = rawVal != 0
		case string:
			parsed, err := strconv.ParseBool(rawVal)
			if err != nil {
				return fmt.Errorf("row %d: important must be one of 0, 1, true, false", i)
			}
			item.Important = parsed
		}

		// Tags
		rawStr, ok = row[7].(string)
		if ok {
			if rawStr != "" {
				// Parse the json
				if err := json.Unmarshal([]byte(rawStr), &item.Tags); err != nil {
					return fmt.Errorf("row %d: tags must be a valid json array of strings", i)
				}
			}
		}

		// Cover
		rawStr, ok = row[6].(string)
		if ok {
			item.Cover = rawStr
		}

		// Created at
		item.Created = readDate(row, 10)

		// Last updated at
		item.LastUpdate = readDate(row, 11)

		// Reminder
		reminderDate := readDate(row, 14)
		if reminderDate != "" {
			item.Reminder = Reminder{
				Date: reminderDate,
			}
		}

		// Collection ID
		rawInt, ok := row[13].(int64)
		if ok {
			if rawInt != 0 {
				item.Collection = Collection{
					Ref: Collections,
					ID:  rawInt,
					OID: rawInt,
				}
			}
		}

		// Set default values
		item.Order = 0
		item.Type = Link

		// Append the item to the request
		request.Items = append(request.Items, item)
	}

	data := &MultipleCreateItemResponse{}

	// Send the request
	res, err := client.R().
		SetHeader("Authorization", fmt.Sprintf("Bearer %s", t.token)).
		SetBody(request).
		SetResult(data).
		Post("https://api.raindrop.io/rest/v1/raindrops")

	if err != nil {
		return fmt.Errorf("failed to insert data: %w", err)
	}

	if res.IsError() {
		return fmt.Errorf("failed to insert data(%d): %s", res.StatusCode(), res.String())
	}

	if !data.Result {
		return fmt.Errorf("failed to insert data: %s", res.String())
	}

	// We need to clear the cache after inserting so that we don't return stale data
	return t.clearCache()

}

// A slice of primary keys to delete
func (t *raindropTable) Delete(primaryKeys []interface{}) error {
	requestBody := &MultipleDeleteItemRequest{}

	for _, key := range primaryKeys {
		if intVal, ok := key.(int64); ok {
			requestBody.IDs = append(requestBody.IDs, intVal)
		}
	}

	data := &MultipleCreateItemResponse{}

	// Send the request
	res, err := client.R().
		SetHeader("Authorization", fmt.Sprintf("Bearer %s", t.token)).
		SetBody(requestBody).
		SetResult(data).
		Delete("https://api.raindrop.io/rest/v1/raindrops/0")

	if err != nil {
		return fmt.Errorf("failed to delete data: %w", err)
	}

	if res.IsError() {
		return fmt.Errorf("failed to delete data(%d): %s", res.StatusCode(), res.String())
	}

	if !data.Result {
		return fmt.Errorf("failed to delete data: %s", res.String())
	}

	// We need to clear the cache after deleting so that we don't return stale data
	return t.clearCache()

}

// A destructor to clean up resources
func (t *raindropTable) Close() error {
	return nil
}

func (t *raindropTable) clearCache() error {
	err := t.db.DropPrefix([]byte("raindrop-"))
	if err != nil {
		return fmt.Errorf("failed to clear cache. You may find stale data. Error: %w", err)
	}
	return nil
}

// Read an interface representing a date from the user
func readDate(rows []interface{}, index int) string {
	if len(rows) <= index {
		return ""
	}
	switch rawVal := rows[index].(type) {
	case string:
		dateFormat := []string{
			time.RFC3339,
			time.DateTime,
			time.TimeOnly,
			time.DateOnly,
			"02/01/2006",
		}
		var t time.Time
		var err error
		for _, format := range dateFormat {
			t, err = time.Parse(format, rawVal)
			if err == nil {
				break
			}
		}
		if err != nil {
			return ""
		}
		return t.Format(time.RFC3339)
	case int64:
		return time.Unix(rawVal, 0).Format(time.RFC3339)
	default:
		return ""
	}

}
