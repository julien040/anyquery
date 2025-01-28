package main

import (
	"crypto/md5"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/julien040/anyquery/rpc"
	"github.com/julien040/anyquery/rpc/helper"
)

type columnInfo struct {
	ID         string
	Type       string
	Name       string
	Index      int
	calculated bool
}

// A constructor to create a new table instance
// This function is called everytime a new connection is made to the plugin
//
// It should return a new table instance, the database schema and if there is an error
func tableCreator(args rpc.TableCreatorArgs) (rpc.Table, *rpc.DatabaseSchema, error) {
	// Example: get a token from the user configuration
	token := args.UserConfig.GetString("token")
	if token == "" {
		return nil, nil, fmt.Errorf("token must be set in the plugin configuration")
	}

	doc_id := args.UserConfig.GetString("doc_id")
	if doc_id == "" {
		return nil, nil, fmt.Errorf("doc_id must be set in the plugin configuration")
	}

	table_id := args.UserConfig.GetString("table_id")
	if table_id == "" {
		return nil, nil, fmt.Errorf("table_id must be set in the plugin configuration")
	}

	hashedToken := sha256.Sum256([]byte(token))
	hashedMD5 := md5.Sum([]byte(token))

	cache, err := helper.NewCache(helper.NewCacheArgs{
		Paths:         []string{"coda", doc_id, table_id, fmt.Sprintf("%x", hashedMD5[:])},
		EncryptionKey: hashedToken[:],
	})

	if err != nil {
		return nil, nil, err
	}

	cols := []rpc.DatabaseSchemaColumn{
		{
			Name:        "id",
			Type:        rpc.ColumnTypeString,
			Description: "The primary key of the row",
		},
	}

	// Request the schema from the API
	columnResponse := &GetColsResp{}
	resp, err := client.R().
		SetHeader("Authorization", fmt.Sprintf("Bearer %s", token)).
		SetPathParams(map[string]string{
			"doc_id":   doc_id,
			"table_id": table_id,
		}).SetResult(columnResponse).
		Get("https://coda.io/apis/v1/docs/{doc_id}/tables/{table_id}/columns")

	if err != nil {
		return nil, nil, fmt.Errorf("error while fetching columns: %w", err)
	}

	if resp.IsError() {
		return nil, nil, fmt.Errorf("error while fetching columns(%d): %s", resp.StatusCode(), resp.String())
	}

	mapColIDToInfo := map[string]columnInfo{}
	for i, col := range columnResponse.Items {
		switch col.Type {
		case "slider", "scale", "number":
			cols = append(cols, rpc.DatabaseSchemaColumn{
				Name:        col.Name,
				Type:        rpc.ColumnTypeFloat,
				Description: fmt.Sprintf("Coda type: %s", col.Type),
			})
		default:
			cols = append(cols, rpc.DatabaseSchemaColumn{
				Name:        col.Name,
				Type:        rpc.ColumnTypeString,
				Description: fmt.Sprintf("Coda type: %s", col.Type),
			})
		}

		info := columnInfo{
			ID:    col.ID,
			Type:  col.Format.Type,
			Name:  col.Name,
			Index: i + 1,
		}
		if col.Calculated != nil {
			info.calculated = *col.Calculated
		}

		mapColIDToInfo[col.ID] = info

	}

	// Request information about the table
	tableResponse := &TableDescription{}
	resp, err = client.R().
		SetHeader("Authorization", fmt.Sprintf("Bearer %s", token)).
		SetPathParams(map[string]string{
			"doc_id":   doc_id,
			"table_id": table_id,
		}).SetResult(tableResponse).
		Get("https://coda.io/apis/v1/docs/{doc_id}/tables/{table_id}")

	if err != nil {
		return nil, nil, fmt.Errorf("error while fetching table information: %w", err)
	}

	if resp.IsError() {
		return nil, nil, fmt.Errorf("error while fetching table information(%d): %s", resp.StatusCode(), resp.String())
	}

	return &codaTable{
			cache:   cache,
			tableID: table_id,
			docID:   doc_id,
			token:   token,
			cols:    mapColIDToInfo,
		}, &rpc.DatabaseSchema{
			HandlesInsert: true,
			HandlesUpdate: true,
			HandlesDelete: true,
			Columns:       cols,
			BufferInsert:  25,
			BufferUpdate:  0, // Coda API doesn't support batch updates
			BufferDelete:  25,
			Description: fmt.Sprintf("Table name in Coda: %s. Created at: %s. Updated at: %s. Broswable at: %s",
				tableResponse.Name, tableResponse.CreatedAt, tableResponse.UpdatedAt, tableResponse.BrowserLink),
		}, nil
}

// The table struct
// There is one per connection to the plugin and is created by the creator function
// In there, you can store any state you need to read the rows (e.g. a database connection, an API token, etc.)
type codaTable struct {
	cache   *helper.Cache
	tableID string
	docID   string
	token   string
	cols    map[string]columnInfo
}

// The cursor struct
// There is one per query and is created by the createReader function
// In there, you can store any state you need to read the rows (e.g. a database connection from codaTable, an offset, a cursor, etc.)
type codaTableCursor struct {
	cache         *helper.Cache
	tableID       string
	docID         string
	token         string
	cols          map[string]columnInfo
	nextPageToken string
	pageID        int
}

// Create a new cursor that will be used to read rows
func (t *codaTable) CreateReader() rpc.ReaderInterface {
	return &codaTableCursor{
		cache:   t.cache,
		tableID: t.tableID,
		docID:   t.docID,
		token:   t.token,
		cols:    t.cols,
	}
}

// Return a slice of rows that will be returned to Anyquery and filtered.
// The second return value is true if the cursor has no more rows to return
//
// The constraints are used for optimization purposes to "pre-filter" the rows
// If the rows returned don't match the constraints, it's not an issue. Anyquery will filter them out
func (t *codaTableCursor) Query(constraints rpc.QueryConstraint) ([][]interface{}, bool, error) {
	// Try to request the next page from the cache
	cacheKey := fmt.Sprintf("%s-%s-%d", t.docID, t.tableID, t.pageID)
	var rows [][]interface{}
	var metadata map[string]interface{}
	var err error
	rows, metadata, err = t.cache.Get(cacheKey)
	if err != nil || len(rows) == 0 { // Cache miss
		// Request the next page from the API
		getRowsRes := &GetRows{}
		req := client.R().
			SetHeader("Authorization", fmt.Sprintf("Bearer %s", t.token)).
			SetQueryParam("limit", "250").
			SetQueryParam("valueFormat", "rich").
			SetPathParams(map[string]string{
				"doc_id":   t.docID,
				"table_id": t.tableID,
			}).SetResult(getRowsRes)

		if t.nextPageToken != "" {
			req.SetQueryParam("pageToken", t.nextPageToken)
		}

		resp, err := req.Get("https://coda.io/apis/v1/docs/{doc_id}/tables/{table_id}/rows")
		if err != nil {
			return nil, false, fmt.Errorf("error while fetching rows: %w", err)
		}

		if resp.IsError() {
			return nil, false, fmt.Errorf("error while fetching rows(%d): %s", resp.StatusCode(), resp.String())
		}

		rows = make([][]interface{}, 0, len(getRowsRes.Items))
		for _, row := range getRowsRes.Items {
			values := make([]interface{}, len(t.cols)+1)
			values[0] = row.ID
			for colID, colInfo := range t.cols {
				if val, ok := row.Values[colID]; ok {
					colValue := interface{}(nil)
					switch parsed := val.(type) {
					case string:
						// Remove the leading and trailing ``` from the string
						colValue = strings.Trim(parsed, "`")
					default:
						colValue = helper.Serialize(val)
					}
					values[colInfo.Index] = colValue
				}
			}
			rows = append(rows, values)
		}

		metadata = map[string]interface{}{
			"nextPageToken": getRowsRes.NextSyncToken,
		}
		t.nextPageToken = getRowsRes.NextPageToken

		// Cache the rows
		err = t.cache.Set(cacheKey, rows, metadata, time.Minute*5)
		if err != nil {
			log.Printf("error while caching rows: %v", err)
		}
		t.pageID++

	} else {
		t.nextPageToken = metadata["nextPageToken"].(string)
		t.pageID++
	}

	return rows, t.nextPageToken == "", nil
}

// We gotta create a custom marshal function to handle the different types of columns
// This is due because for some types, the user might return the JSON LD representation or just a string/number
func marshal(val interface{}, Type string) interface{} {
	switch Type {
	case "packObject", "button", "formula", "attachments":
		return nil
	case "currency":
		// Check if map of interface
		// SQLite will return a JSON LD representation of the value as a string
		// We need to parse it to get the amount
		// It it fails, we consider the user directly inputted the amount at err != nil
		if v, ok := val.(string); ok {
			mapVal := map[string]interface{}{}
			err := json.Unmarshal([]byte(v), &mapVal)
			if err != nil {
				return v
			}
			if amount, ok := mapVal["amount"]; ok {
				return amount
			}
		} else if v, ok := val.(map[string]interface{}); ok {
			if amount, ok := v["amount"]; ok {
				return amount
			}
		}
	case "lookup":
		// Check if map of interface
		if v, ok := val.(string); ok {
			mapVal := map[string]interface{}{}
			err := json.Unmarshal([]byte(v), &mapVal)
			if err != nil {
				return v
			}
			if name, ok := mapVal["name"]; ok {
				return name
			}
		} else if v, ok := val.(map[string]interface{}); ok {
			if name, ok := v["name"]; ok {
				return name
			}
		}
	case "image":
		// Check if map of interface
		if v, ok := val.(string); ok {
			mapVal := map[string]interface{}{}
			err := json.Unmarshal([]byte(v), &mapVal)
			if err != nil {
				return v
			}
			if url, ok := mapVal["url"]; ok {
				return url
			}
		} else if v, ok := val.(map[string]interface{}); ok {
			if url, ok := v["url"]; ok {
				return url
			}
		}
	case "person":
		// Check if map of interface
		if v, ok := val.(string); ok {
			mapVal := map[string]interface{}{}
			err := json.Unmarshal([]byte(v), &mapVal)
			if err != nil {
				return v
			}
			if name, ok := mapVal["name"]; ok {
				return name
			}
		} else if v, ok := val.(map[string]interface{}); ok {
			if name, ok := v["name"]; ok {
				return name
			}
		}
	case "hyperlink":
		// Check if map of interface
		if v, ok := val.(string); ok {
			mapVal := map[string]interface{}{}
			err := json.Unmarshal([]byte(v), &mapVal)
			if err != nil {
				return v
			}
			if url, ok := mapVal["url"]; ok {
				return url
			}
		} else if v, ok := val.(map[string]interface{}); ok {
			if url, ok := v["url"]; ok {
				return url
			}
		}
	case "percent":
		switch v := val.(type) {
		case string:
			return v
		case float64:
			return v
		case int64:
			return v * 10 // No idea why it's not 100
		}
	}

	return val

}

// A slice of rows to insert
func (t *codaTable) Insert(rows [][]interface{}) error {
	// Example: insert the rows in a database
	// for _, row := range rows {
	// 	err := db.Insert(row[0], row[1], row[2])
	// 	if err != nil {
	// 		return err
	// 	}

	requestBody := InsertRowsBody{}
	requestBody.Rows = make([]InsertRow, 0, len(rows))
	for _, row := range rows {
		cells := []InsertCell{}
		for i, col := range row {
			// Skip the primary key
			if i == 0 {
				continue
			}

			// To avoid inserting nil values
			if col == nil {
				continue
			}

			for colID, colInfo := range t.cols {
				if colInfo.Index == i {
					marshalled := marshal(col, colInfo.Type)
					if marshalled == nil {
						// This value is incompatible
						break
					}
					if colInfo.calculated {
						// Skip calculated columns
						break
					}
					cells = append(cells, InsertCell{
						Column: colID,
						Value:  marshalled,
					})
					break
				}
			}
		}
		// If no cells were added, skip this row
		if len(cells) == 0 {
			continue
		}
		requestBody.Rows = append(requestBody.Rows, InsertRow{
			Cells: cells,
		})
	}

	resp, err := client.R().
		SetHeader("Authorization", fmt.Sprintf("Bearer %s", t.token)).
		SetBody(requestBody).
		SetPathParams(map[string]string{
			"doc_id":   t.docID,
			"table_id": t.tableID,
		}).
		Post("https://coda.io/apis/v1/docs/{doc_id}/tables/{table_id}/rows")

	if err != nil {
		return fmt.Errorf("error while inserting rows: %w", err)
	}

	if resp.IsError() {
		return fmt.Errorf("error while inserting rows(%d): %s", resp.StatusCode(), resp.String())
	}

	// Clear the cache
	err = t.cache.Clear()
	if err != nil {
		log.Printf("error while clearing cache: %v", err)
	}

	return nil
}

// A slice of rows to update
// The first element of each row is the primary key
// while the rest are the values to update
// The primary key is therefore present twice
func (t *codaTable) Update(rows [][]interface{}) error {
	for _, row := range rows {
		cells := []InsertCell{}
		for i, col := range row {
			// Skip both primary key
			if i < 2 {
				continue
			}

			// To avoid inserting nil values
			if col == nil {
				continue
			}

			for colID, colInfo := range t.cols {
				if colInfo.Index == i-1 { // -1 because we skip the primary key

					marshalled := marshal(col, colInfo.Type)
					if marshalled == nil {
						// This value is incompatible
						break
					}
					if colInfo.calculated {
						// Skip calculated columns
						break
					}
					cells = append(cells, InsertCell{
						Column: colID,
						Value:  marshalled,
					})
				}
			}
		}
		// Update the row to the API
		updateRow := &UpdateRowsBody{
			Row: UpdateRow{
				Cells: cells,
			},
		}

		resp, err := client.R().
			SetHeader("Authorization", fmt.Sprintf("Bearer %s", t.token)).
			SetBody(updateRow).
			SetPathParams(map[string]string{
				"doc_id":   t.docID,
				"table_id": t.tableID,
				"row_id":   row[0].(string),
			}).
			Put("https://coda.io/apis/v1/docs/{doc_id}/tables/{table_id}/rows/{row_id}")

		if err != nil {
			return fmt.Errorf("error while updating rows: %w", err)
		}

		if resp.IsError() {
			return fmt.Errorf("error while updating rows(%d): %s", resp.StatusCode(), resp.String())
		}

		// Clear the cache
		err = t.cache.Clear()
		if err != nil {
			log.Printf("error while clearing cache: %v", err)
		}
	}

	return nil
}

// A slice of primary keys to delete
func (t *codaTable) Delete(primaryKeys []interface{}) error {
	body := DeleteRowBody{
		RowIds: make([]string, 0, len(primaryKeys)),
	}
	for _, pk := range primaryKeys {
		if parsed, ok := pk.(string); ok {
			body.RowIds = append(body.RowIds, parsed)
		}
	}

	resp, err := client.R().
		SetHeader("Authorization", fmt.Sprintf("Bearer %s", t.token)).
		SetBody(body).
		SetPathParams(map[string]string{
			"doc_id":   t.docID,
			"table_id": t.tableID,
		}).
		Delete("https://coda.io/apis/v1/docs/{doc_id}/tables/{table_id}/rows")

	if err != nil {
		return fmt.Errorf("error while deleting rows: %w", err)
	}

	if resp.IsError() {
		return fmt.Errorf("error while deleting rows(%d): %s", resp.StatusCode(), resp.String())
	}

	// Clear the cache
	err = t.cache.Clear()
	if err != nil {
		log.Printf("error while clearing cache: %v", err)
	}

	return nil
}

// A destructor to clean up resources
func (t *codaTable) Close() error {
	return nil
}
