package main

import (
	"bytes"
	"crypto/md5"
	"encoding/gob"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path"
	"strconv"
	"time"

	"github.com/adrg/xdg"
	"github.com/dgraph-io/badger/v4"
	"github.com/go-resty/resty/v2"
	"github.com/hashicorp/go-retryablehttp"
	"github.com/julien040/anyquery/rpc"
)

var retryableClient = retryablehttp.NewClient()

var client = resty.NewWithClient(retryableClient.StandardClient())

// A constructor to create a new table instance
// This function is called everytime a new connection is made to the plugin
//
// It should return a new table instance, the database schema and if there is an error
func itemsCreator(args rpc.TableCreatorArgs) (rpc.Table, *rpc.DatabaseSchema, error) {
	var token, consumerKey string

	// Get the token and consumer key from the arguments
	if _, ok := args.UserConfig["token"]; !ok {
		return nil, nil, fmt.Errorf("missing token in user config")
	} else {
		rawToken, ok := args.UserConfig["token"].(string)
		if !ok {
			return nil, nil, fmt.Errorf("token must be a string")
		}
		token = rawToken
	}

	if _, ok := args.UserConfig["consumer_key"]; !ok {
		return nil, nil, fmt.Errorf("missing consumer_key in user config")
	} else {
		rawConsumerKey, ok := args.UserConfig["consumer_key"].(string)
		if !ok {
			return nil, nil, fmt.Errorf("consumer_key must be a string")
		}
		consumerKey = rawConsumerKey
	}

	// Hash the token to create a folder name
	hashedToken := md5.Sum([]byte(token))

	cacheFolder := path.Join(xdg.CacheHome, "anyquery", "plugins", "pocket", fmt.Sprintf("%x", hashedToken))
	err := os.MkdirAll(cacheFolder, 0755)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create cache folder: %w", err)
	}

	// Open the badger database encrypted with the toke
	options := badger.DefaultOptions(cacheFolder).WithEncryptionKey(hashedToken[:]).
		WithNumVersionsToKeep(1).WithCompactL0OnClose(true).WithValueLogFileSize(2 << 22).
		WithIndexCacheSize(2 << 22)
	db, err := badger.Open(options)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to open badger database: %w", err)
	}

	return &itemsTable{
			consumerKey: consumerKey,
			accessToken: token,
			db:          db,
		}, &rpc.DatabaseSchema{
			BufferInsert:  100,
			BufferDelete:  100,
			HandlesInsert: true,
			HandlesUpdate: false,
			HandlesDelete: true,
			Columns: []rpc.DatabaseSchemaColumn{
				{
					Name: "id",
					Type: rpc.ColumnTypeString,
				},
				{
					Name: "given_url",
					Type: rpc.ColumnTypeString,
				},
				{
					Name: "given_title",
					Type: rpc.ColumnTypeString,
				},
				{
					Name: "resolved_url",
					Type: rpc.ColumnTypeString,
				},
				{
					Name: "resolved_title",
					Type: rpc.ColumnTypeString,
				},
				{
					Name: "excerpt",
					Type: rpc.ColumnTypeString,
				},
				{
					Name: "lang",
					Type: rpc.ColumnTypeString,
				},
				{
					Name: "favorite",
					Type: rpc.ColumnTypeInt,
				},
				{
					Name: "status",
					Type: rpc.ColumnTypeInt,
				},
				{
					Name: "time_added",
					Type: rpc.ColumnTypeInt,
				},
				{
					Name: "time_updated",
					Type: rpc.ColumnTypeInt,
				},
				{
					Name: "time_favorited",
					Type: rpc.ColumnTypeInt,
				},
				{
					Name: "time_read",
					Type: rpc.ColumnTypeInt,
				},
				{
					Name: "is_article",
					Type: rpc.ColumnTypeInt,
				},
				{
					Name: "has_image",
					Type: rpc.ColumnTypeInt,
				},
				{
					Name: "has_video",
					Type: rpc.ColumnTypeInt,
				},
				{
					Name: "word_count",
					Type: rpc.ColumnTypeInt,
				},
				{
					Name: "time_to_read",
					Type: rpc.ColumnTypeInt,
				},
				{
					Name: "listen_duration_estimate",
					Type: rpc.ColumnTypeInt,
				},
			},
		}, nil
}

type itemsTable struct {
	consumerKey string
	accessToken string
	db          *badger.DB
}

type itemsCursor struct {
	consumerKey string
	accessToken string
	db          *badger.DB
	offset      int
	pageSize    int
}

// Return a slice of rows that will be returned to Anyquery and filtered.
// The second return value is true if the cursor has no more rows to return
//
// The constraints are used for optimization purposes to "pre-filter" the rows
// If the rows returned don't match the constraints, it's not an issue. Anyquery will filter them out
func (t *itemsCursor) Query(constraints rpc.QueryConstraint) ([][]interface{}, bool, error) {
	// Cache key
	key := fmt.Sprintf("items-%d-%d", t.offset, t.pageSize)

	// Get the rows from the cache if they exist
	var rows [][]interface{}

	err := t.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte(key))
		if err != nil {
			return err
		}
		item.Value(func(val []byte) error {
			decoder := gob.NewDecoder(bytes.NewReader(val))
			err := decoder.Decode(&rows)
			if err != nil {
				return err
			}
			return nil
		})
		return nil
	})
	if err == nil {
		// If the rows are in the cache, return them
		t.offset += t.pageSize
		return rows, len(rows) < int(t.pageSize), nil
	}

	response := &retrieveResponse{}

	// Get the items from the Pocket API
	res, err := client.R().SetBody(&retrieveRequest{
		ConsumerKey: t.consumerKey,
		AccessToken: t.accessToken,
		Count:       t.pageSize,
		Offset:      t.offset,
		State:       retrieveAll,
		Sort:        sortNewest,
	}).SetResult(response).Post("https://getpocket.com/v3/get")

	if err != nil {
		log.Printf("failed to retrieve items from Pocket API: %v %s", err, res.String())
		return nil, true, fmt.Errorf("failed to retrieve items from Pocket API")
	}

	if res.StatusCode() != 200 {
		log.Printf("failed to retrieve items from Pocket API: %v %s", res.StatusCode(), res.String())
		return nil, true, fmt.Errorf("pocket API returned http status code %d", res.StatusCode())
	}

	if response.Status != 1 {
		log.Printf("failed to retrieve items from Pocket API: %v (error %s)", response, res.RawResponse.Header["X-Error"])
		return nil, true, fmt.Errorf("code %d from Pocket API", response.Status)
	}

	rows = make([][]interface{}, 0, len(response.List))

	for k, item := range response.List {
		favorite, _ := strconv.Atoi(item.Favorite)
		status, _ := strconv.Atoi(item.Status)
		timeAdded, _ := strconv.Atoi(item.TimeAdded)
		timeUpdated, _ := strconv.Atoi(item.TimeUpdated)
		timeFavorited, _ := strconv.Atoi(item.TimeFavorited)
		timeRead, _ := strconv.Atoi(item.TimeRead)
		isArticle, _ := strconv.Atoi(item.IsArticle)
		hasImage, _ := strconv.Atoi(item.HasImage)
		hasVideo, _ := strconv.Atoi(item.HasVideo)
		wordCount, _ := strconv.Atoi(item.WordCount)
		timeToRead := item.TimeToRead

		timeAddedParsed := interface{}(nil)
		if timeAdded > 0 {
			timeAddedParsed = time.Unix(int64(timeAdded), 0).Format(time.RFC3339)
		}

		timeUpdatedParsed := interface{}(nil)
		if timeUpdated > 0 {
			timeUpdatedParsed = time.Unix(int64(timeUpdated), 0).Format(time.RFC3339)
		}

		timeFavoritedParsed := interface{}(nil)
		if timeFavorited > 0 {
			timeFavoritedParsed = time.Unix(int64(timeFavorited), 0).Format(time.RFC3339)
		}

		timeReadParsed := interface{}(nil)
		if timeRead > 0 {
			timeReadParsed = time.Unix(int64(timeRead), 0).Format(time.RFC3339)
		}

		rows = append(rows, []interface{}{
			k,
			item.GivenURL,
			item.GivenTitle,
			item.ResolvedURL,
			item.ResolvedTitle,
			item.Excerpt,
			item.Lang,
			favorite,
			status,
			timeAddedParsed,
			timeUpdatedParsed,
			timeFavoritedParsed,
			timeReadParsed,
			isArticle,
			hasImage,
			hasVideo,
			wordCount,
			timeToRead,
			item.ListenDurationEstimate,
		})
	}

	// Save the rows to the cache
	err = t.db.Update(func(txn *badger.Txn) error {
		var buf bytes.Buffer
		encoder := gob.NewEncoder(&buf)
		err := encoder.Encode(rows)
		if err != nil {
			return err
		}
		e := badger.NewEntry([]byte(key), buf.Bytes()).WithTTL(time.Hour)
		err = txn.SetEntry(e)
		return err
	})

	if err != nil {
		log.Printf("failed to save items to cache: %v", err)
	}
	t.offset += t.pageSize

	return rows, len(rows) < int(t.pageSize), nil
}

// Create a new cursor that will be used to read rows
func (t *itemsTable) CreateReader() rpc.ReaderInterface {
	return &itemsCursor{
		consumerKey: t.consumerKey,
		accessToken: t.accessToken,
		db:          t.db,
		offset:      0,
		pageSize:    100,
	}
}

// A slice of rows to insert
func (t *itemsTable) Insert(rows [][]interface{}) error {
	actions := []interface{}{}
	for i, row := range rows {
		// Find the URL
		url := ""
		rawVal, ok := row[1].(string)
		if ok {
			url = rawVal
		} else {
			rawVal, ok := row[3].(string)
			if ok {
				url = rawVal
			} else {
				return fmt.Errorf("missing URL in row %d", i)
			}
		}

		// Find the title
		title := ""
		rawVal, ok = row[2].(string)
		if ok {
			title = rawVal
		} else {
			rawVal, ok := row[4].(string)
			if ok {
				title = rawVal
			}
		}

		actions = append(actions, &addAction{
			Action: "add",
			URL:    url,
			Title:  title,
		})
	}

	// Serialize the actions as JSON
	actionsJSON, err := json.Marshal(actions)
	if err != nil {
		return fmt.Errorf("failed to serialize actions: %w", err)
	}

	result := &actionResponse{}

	res, err := client.R().SetQueryParams(map[string]string{}).SetBody(&map[string]string{
		"actions":      string(actionsJSON),
		"access_token": t.accessToken,
		"consumer_key": t.consumerKey,
	}).SetResult(result).Post("https://getpocket.com/v3/send")
	if err != nil {
		return fmt.Errorf("failed to send actions to Pocket API: %w. Response: %s", err, res.String())
	}

	if res.StatusCode() != 200 {
		return fmt.Errorf("failed to send actions to Pocket API: http status code %d. Res %s. Error %s", res.StatusCode(), res.String(),
			res.Header()["X-Error"])
	}

	if result.Status != 1 {
		return fmt.Errorf("failed to send actions to Pocket API: %#v", result.ActionErrors)
	} else {
		clearCache(t.db)
	}

	return nil
}

// A slice of rows to update
// The first element of each row is the primary key
// while the rest are the values to update
// The primary key is therefore present twice
func (t *itemsTable) Update(rows [][]interface{}) error {
	return nil
}

// A slice of primary keys to delete
func (t *itemsTable) Delete(primaryKeys []interface{}) error {
	actions := []interface{}{}
	for _, primaryKey := range primaryKeys {
		actions = append(actions, &deleteAction{
			Action: "delete",
			ItemID: primaryKey.(string),
		})
	}

	// Serialize the actions as JSON
	actionsJSON, err := json.Marshal(actions)
	if err != nil {
		return fmt.Errorf("failed to serialize actions: %w", err)
	}

	result := &actionResponse{}

	res, err := client.R().SetBody(&map[string]string{
		"actions":      string(actionsJSON),
		"access_token": t.accessToken,
		"consumer_key": t.consumerKey,
	}).SetResult(result).Post("https://getpocket.com/v3/send")
	if err != nil {
		return fmt.Errorf("failed to send actions to Pocket API: %w. Response: %s", err, res.String())
	}

	if res.StatusCode() != 200 {
		return fmt.Errorf("failed to send actions to Pocket API: http status code %d. Res %s. Error %s", res.StatusCode(), res.String(), res.Header()["X-Error"])
	}

	if result.Status != 1 {
		return fmt.Errorf("failed to send actions to Pocket API: %#v", result.ActionErrors)
	} else {
		clearCache(t.db)
	}

	return nil
}

// A destructor to clean up resources
func (t *itemsTable) Close() error {
	return nil
}

func clearCache(db *badger.DB) {
	err := db.DropAll()
	if err != nil {
		log.Printf("failed to clear cache: %v", err)
	}
}
