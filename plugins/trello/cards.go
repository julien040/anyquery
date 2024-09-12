package main

import (
	"crypto/md5"
	"crypto/sha256"
	"encoding/json"
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
func cardsCreator(args rpc.TableCreatorArgs) (rpc.Table, *rpc.DatabaseSchema, error) {
	key := args.UserConfig.GetString("key")
	token := args.UserConfig.GetString("token")

	if key == "" || token == "" {
		return nil, nil, fmt.Errorf("key and token must be set in the plugin configuration to non-empty values")
	}

	hashedToken := md5.Sum([]byte(token))
	sha256tok := sha256.Sum256([]byte(token))

	// Open the cache
	cache, err := helper.NewCache(helper.NewCacheArgs{
		Paths:         []string{"trello", "cards", fmt.Sprintf("%x", sha256tok[:16])},
		EncryptionKey: []byte(fmt.Sprintf("%x", hashedToken[:16])),
	})

	if err != nil {
		return nil, nil, fmt.Errorf("unable to open the cache %w", err)
	}

	return &cardsTable{
			cache: cache,
			key:   key,
			token: token,
		}, &rpc.DatabaseSchema{
			HandlesInsert: true,
			HandlesUpdate: true,
			HandlesDelete: true,
			PrimaryKey:    2,
			Columns: []rpc.DatabaseSchemaColumn{
				{
					Name:        "board_id",
					Type:        rpc.ColumnTypeString,
					IsParameter: true,
					IsRequired:  false, // So that you can directly use INSERT INTO trello_cards('list_id') VALUES ('id')
				},
				{
					Name: "list_id",
					Type: rpc.ColumnTypeString,
				},
				{
					Name: "id",
					Type: rpc.ColumnTypeString,
				},
				{
					Name: "name",
					Type: rpc.ColumnTypeString,
				},
				{
					Name: "description",
					Type: rpc.ColumnTypeString,
				},
				{
					Name: "position",
					Type: rpc.ColumnTypeInt,
				},
				{
					Name: "url",
					Type: rpc.ColumnTypeString,
				},
				{
					Name: "start_at",
					Type: rpc.ColumnTypeString,
				},
				{
					Name: "due_at",
					Type: rpc.ColumnTypeString,
				},
				{
					Name: "due_completed",
					Type: rpc.ColumnTypeString,
				},
				{
					Name: "due_reminder",
					Type: rpc.ColumnTypeString,
				},
				{
					Name: "comments_count",
					Type: rpc.ColumnTypeInt,
				},
				{
					Name: "votes_count",
					Type: rpc.ColumnTypeInt,
				},
				{
					Name: "checklists_count",
					Type: rpc.ColumnTypeInt,
				},
				{
					Name: "checked_items_count",
					Type: rpc.ColumnTypeInt,
				},
				{
					Name: "attachments_count",
					Type: rpc.ColumnTypeInt,
				},
				{
					Name: "labels",
					Type: rpc.ColumnTypeString,
				},
				{
					Name: "subscribed",
					Type: rpc.ColumnTypeBool,
				},
				{
					Name: "location",
					Type: rpc.ColumnTypeString,
				},
				{
					Name: "member_ids",
					Type: rpc.ColumnTypeString,
				},
				{
					Name: "label_ids",
					Type: rpc.ColumnTypeString,
				},
			},
		}, nil
}

type cardsTable struct {
	cache *helper.Cache
	key   string
	token string
}

type cardsCursor struct {
	cache  *helper.Cache
	key    string
	token  string
	cursor string
}

// Return a slice of rows that will be returned to Anyquery and filtered.
// The second return value is true if the cursor has no more rows to return
//
// The constraints are used for optimization purposes to "pre-filter" the rows
// If the rows returned don't match the constraints, it's not an issue. Anyquery will filter them out
func (t *cardsCursor) Query(constraints rpc.QueryConstraint) ([][]interface{}, bool, error) {

	boardID := constraints.GetColumnConstraint(0).GetStringValue()
	if boardID == "" {
		return nil, true, fmt.Errorf("board_id must be set. To do so, use the following query: SELECT * FROM trello_cards('board_id');")
	}

	cacheKey := fmt.Sprintf("%s-%s", boardID, t.cursor)

	// Try to get the data from the cache
	data, metadata, err := t.cache.Get(cacheKey)
	if err == nil {
		// Save the cursor for the next iteration
		t.cursor = metadata["cursor"].(string)
		return data, len(data) < 1000, nil
	}

	// If the cache is empty, fetch the data from the API
	endpoint := "https://api.trello.com/1/boards/{boardID}/cards"
	body := Cards{}

	req := client.R().SetPathParam("boardID", boardID).
		SetQueryParams(map[string]string{
			"key":   t.key,
			"token": t.token,
		}).SetResult(&body)

	if t.cursor != "" {
		req.SetQueryParam("before", t.cursor)
	}

	resp, err := req.Get(endpoint)
	if err != nil {
		return nil, true, fmt.Errorf("unable to fetch data from the API %w", err)
	}

	if resp.IsError() {
		return nil, true, fmt.Errorf("unable to fetch data from the API (%d): %s", resp.StatusCode(), resp.String())
	}

	// Save the cursor for the next iteration
	if len(body) > 0 {
		t.cursor = body[0].ID
	} else {
		t.cursor = ""
	}

	// Compute the rows
	rows := make([][]interface{}, 0, len(body))
	for _, card := range body {
		dueAt := interface{}(nil)
		if card.Due != nil {
			dueAt = *card.Due
		}
		// The API returns the number of minutes before the due date to remind the user
		// We need to compute the actual reminder date
		dueReminder := interface{}(nil)
		if card.DueReminder != nil {
			dueAtParsed, err := time.Parse(time.RFC3339, *card.Due)
			if err == nil {
				dueAtParsed = dueAtParsed.Add(-time.Duration((*card.DueReminder)) * time.Minute)
				dueReminder = dueAtParsed.Format(time.RFC3339)
			}
		}

		startAt := interface{}(nil)
		if card.Start != nil {
			startAt = *card.Start
		}

		labels := []string{}
		for _, label := range card.Labels {
			labels = append(labels, label.Name)
		}
		rows = append(rows, []interface{}{
			card.IDList,
			card.ID,
			card.Name,
			card.Desc,
			card.Pos,
			card.URL,
			startAt,
			dueAt,
			card.DueComplete,
			dueReminder,
			card.Badges.Comments,
			card.Badges.Votes,
			card.Badges.CheckItems,
			card.Badges.CheckItemsChecked,
			card.Badges.Attachments,
			labels,
			card.Subscribed,
			card.Badges.Location,
			card.IDMembers,
			card.IDLabels,
		})
	}

	// Save the data in the cache
	err = t.cache.Set(cacheKey, rows, map[string]interface{}{
		"cursor": t.cursor,
	}, 5*time.Minute)

	if err != nil {
		log.Printf("unable to save the data in the cache %v", err)
	}

	return rows, len(body) < 1000 || t.cursor == "", nil
}

// Create a new cursor that will be used to read rows
func (t *cardsTable) CreateReader() rpc.ReaderInterface {
	return &cardsCursor{
		cache: t.cache,
		key:   t.key,
		token: t.token,
	}
}

type insertBody struct {
	IDList      string   `json:"idList,omitempty"`
	Name        string   `json:"name,omitempty"`
	Desc        string   `json:"desc,omitempty"`
	Pos         int64    `json:"pos,omitempty"`
	Start       string   `json:"start,omitempty"`
	Due         string   `json:"due,omitempty"`
	DueComplete bool     `json:"dueComplete,omitempty"`
	Subscribed  bool     `json:"subscribed,omitempty"`
	IDMembers   []string `json:"idMembers,omitempty"`
	IDLabels    []string `json:"idLabels,omitempty"`
}

// A slice of rows to insert
func (t *cardsTable) Insert(rows [][]interface{}) error {
	endpoint := "https://api.trello.com/1/cards"
	for _, row := range rows {
		reqBody := insertBody{}
		if len(row) != 21 {
			return fmt.Errorf("expected 21 columns, got %d", len(row))
		}

		var ok bool
		if reqBody.IDList, ok = row[1].(string); !ok {
			return fmt.Errorf("expected a string for idList, got %T", row[0])
		}

		if reqBody.IDList == "" {
			return fmt.Errorf("list_id must be set")
		}

		if row[3] != nil {
			if reqBody.Name, ok = row[3].(string); !ok {
				return fmt.Errorf("expected a string for name, got %T", row[3])
			}
		}

		if row[4] != nil {
			if reqBody.Desc, ok = row[4].(string); !ok {
				return fmt.Errorf("expected a string for desc, got %T", row[4])
			}
		}

		if row[5] != nil {
			if reqBody.Pos, ok = row[5].(int64); !ok {
				return fmt.Errorf("expected an int for pos, got %T", row[5])
			}
		}

		if row[7] != nil {
			if reqBody.Start, ok = row[7].(string); !ok {
				return fmt.Errorf("expected a string for start, got %T", row[7])
			}
			// Parse the time
			parsed, err := parseTime(reqBody.Start)
			if err != nil {
				return fmt.Errorf("unable to parse the time %w", err)
			}
			reqBody.Start = parsed.Format(time.RFC3339)
		}

		if row[8] != nil {
			if reqBody.Due, ok = row[8].(string); !ok {
				return fmt.Errorf("expected a string for due, got %T", row[8])
			}
			// Parse the time
			parsed, err := parseTime(reqBody.Due)
			if err != nil {
				return fmt.Errorf("unable to parse the time %w", err)
			}
			reqBody.Due = parsed.Format(time.RFC3339)

		}

		if row[9] != nil {
			if boolTemp, ok := row[9].(int64); !ok {
				return fmt.Errorf("expected a bool for dueComplete, got %T", row[9])
			} else {
				reqBody.DueComplete = boolTemp == 1
			}
		}

		if row[17] != nil {
			if boolTemp, ok := row[17].(int64); !ok {
				return fmt.Errorf("expected a bool for subscribed, got %T", row[17])
			} else {
				reqBody.Subscribed = boolTemp == 1
			}
		}

		if row[19] != nil {
			if idMembers, ok := row[19].(string); !ok {
				return fmt.Errorf("expected a string for idMembers, got %T", row[19])
			} else {
				err := json.Unmarshal([]byte(idMembers), &reqBody.IDMembers)
				if err != nil {
					return fmt.Errorf("idMembers must be a valid JSON array %w", err)
				}
			}
		}

		if row[20] != nil {
			if idLabels, ok := row[20].(string); !ok {
				return fmt.Errorf("expected a string for idLabels, got %T", row[20])
			} else {
				err := json.Unmarshal([]byte(idLabels), &reqBody.IDLabels)
				if err != nil {
					return fmt.Errorf("idLabels must be a valid JSON array %w", err)
				}
			}
		}

		resp, err := client.R().SetBody(reqBody).
			SetQueryParam("key", t.key).
			SetQueryParam("token", t.token).
			Post(endpoint)

		if err != nil {
			return fmt.Errorf("unable to fetch data from the API %w", err)
		}

		if resp.IsError() {
			return fmt.Errorf("unable to fetch data from the API (%d): %s", resp.StatusCode(), resp.String())
		}

	}
	return nil
}

// A slice of rows to update
// The first element of each row is the primary key
// while the rest are the values to update
// The primary key is therefore present twice
func (t *cardsTable) Update(rows [][]interface{}) error {
	endpoint := "https://api.trello.com/1/cards/{cardID}"
	for _, row := range rows {
		trelloID := row[0].(string)

		reqBody := insertBody{}
		if len(row) != 22 {
			return fmt.Errorf("expected 21 columns, got %d", len(row))
		}
		var ok bool
		if row[2] != nil {
			if reqBody.IDList, ok = row[2].(string); !ok {
				return fmt.Errorf("expected a string for list_id, got %T", row[2])
			}
		}

		if row[4] != nil {
			if reqBody.Name, ok = row[4].(string); !ok {
				return fmt.Errorf("expected a string for name, got %T", row[4])
			}
		}

		if row[5] != nil {
			if reqBody.Desc, ok = row[5].(string); !ok {
				return fmt.Errorf("expected a string for desc, got %T", row[5])
			}
		}

		if row[6] != nil {
			if reqBody.Pos, ok = row[6].(int64); !ok {
				return fmt.Errorf("expected an int for pos, got %T", row[6])
			}
		}

		if row[8] != nil {
			if reqBody.Start, ok = row[8].(string); !ok {
				return fmt.Errorf("expected a string for start, got %T", row[8])
			}
			// Parse the time
			parsed, err := parseTime(reqBody.Start)
			if err != nil {
				return fmt.Errorf("unable to parse the time %w", err)
			}
			reqBody.Start = parsed.Format(time.RFC3339)
		}

		if row[9] != nil {
			if reqBody.Due, ok = row[9].(string); !ok {
				return fmt.Errorf("expected a string for due, got %T", row[9])
			}
			// Parse the time
			parsed, err := parseTime(reqBody.Due)
			if err != nil {
				return fmt.Errorf("unable to parse the time %w", err)
			}
			reqBody.Due = parsed.Format(time.RFC3339)

		}

		if row[10] != nil {
			if boolTemp, ok := row[10].(int64); !ok {
				return fmt.Errorf("expected a bool for dueComplete, got %T", row[10])
			} else {
				reqBody.DueComplete = boolTemp == 1
			}
		}

		if row[18] != nil {
			if boolTemp, ok := row[16].(int64); !ok {
				return fmt.Errorf("expected a bool for subscribed, got %T", row[16])
			} else {
				reqBody.Subscribed = boolTemp == 1
			}
		}

		if row[20] != nil {
			if idMembers, ok := row[20].(string); !ok {
				return fmt.Errorf("expected a string for idMembers, got %T", row[20])
			} else {
				err := json.Unmarshal([]byte(idMembers), &reqBody.IDMembers)
				if err != nil {
					return fmt.Errorf("idMembers must be a valid JSON array %w", err)
				}
			}
		}

		if row[21] != nil {
			if idLabels, ok := row[21].(string); !ok {
				return fmt.Errorf("expected a string for idLabels, got %T", row[21])
			} else {
				err := json.Unmarshal([]byte(idLabels), &reqBody.IDLabels)
				if err != nil {
					return fmt.Errorf("idLabels must be a valid JSON array %w", err)
				}
			}
		}

		resp, err := client.R().SetBody(reqBody).
			SetQueryParam("key", t.key).
			SetQueryParam("token", t.token).
			SetPathParam("cardID", trelloID).
			Put(endpoint)

		if err != nil {
			return fmt.Errorf("unable to fetch data from the API %w", err)
		}

		if resp.IsError() {
			return fmt.Errorf("unable to fetch data from the API (%d): %s", resp.StatusCode(), resp.String())
		}

	}
	return nil
}

// A slice of primary keys to delete
func (t *cardsTable) Delete(primaryKeys []interface{}) error {
	endpoint := "https://api.trello.com/1/cards/{cardID}"
	for _, key := range primaryKeys {
		trelloID := key.(string)

		res, err := client.R().
			SetQueryParam("key", t.key).
			SetQueryParam("token", t.token).
			SetPathParam("cardID", trelloID).
			Delete(endpoint)

		if err != nil {
			return fmt.Errorf("unable to fetch data from the API %w", err)
		}

		if res.IsError() {
			return fmt.Errorf("unable to fetch data from the API (%d): %s", res.StatusCode(), res.String())
		}
	}
	return nil
}

// A destructor to clean up resources
func (t *cardsTable) Close() error {
	return t.cache.Close()
}

func parseTime(timeStr string) (time.Time, error) {
	format := []string{time.RFC3339, time.DateTime, time.DateOnly}
	for _, ft := range format {
		t, err := time.Parse(ft, timeStr)
		if err == nil {
			return t, nil
		}
	}

	return time.Time{}, fmt.Errorf("unable to parse the time %s", timeStr)
}
