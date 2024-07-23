package main

import (
	"bytes"
	"crypto/md5"
	"encoding/gob"
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"os"
	"path"
	"time"

	"github.com/adrg/xdg"
	"github.com/dgraph-io/badger/v4"
	"github.com/go-resty/resty/v2"
	"github.com/hashicorp/go-retryablehttp"
	"github.com/julien040/anyquery/rpc"
	"go.uber.org/ratelimit"
)

const getSchemaURL = "https://api.airtable.com/v0/meta/bases/{baseId}/tables"
const tableURL = "https://api.airtable.com/v0/{baseId}/{tableId}"

var retryClient = retryablehttp.NewClient()
var client = resty.NewWithClient(retryClient.StandardClient())

type column struct {
	index   int
	name    string
	colType string
}

// A constructor to create a new table instance
// This function is called everytime a new connection is made to the plugin
//
// It should return a new table instance, the database schema and if there is an error
func (t *tablePlugin) tableCreator(args rpc.TableCreatorArgs) (rpc.Table, *rpc.DatabaseSchema, error) {
	// Get the user configuration
	var token, base, table string
	var cacheEnabled bool
	if rawInter, ok := args.UserConfig["token"]; ok {
		if token, ok = rawInter.(string); !ok {
			return nil, nil, fmt.Errorf("token must be a string")
		}
	}
	if rawInter, ok := args.UserConfig["base"]; ok {
		if base, ok = rawInter.(string); !ok {
			return nil, nil, fmt.Errorf("base must be a string")
		}
	}
	if rawInter, ok := args.UserConfig["table"]; ok {
		if table, ok = rawInter.(string); !ok {
			return nil, nil, fmt.Errorf("table must be a string")
		}
	}
	if rawInter, ok := args.UserConfig["cache"]; ok {
		cacheEnabled, _ = rawInter.(bool)
	}

	// Register the rate limiter if it doesn't exist
	if _, ok := t.rateLimiter[base]; !ok {
		t.mapMutex.Lock()
		t.rateLimiter[base] = ratelimit.New(5)
		t.mapMutex.Unlock()
	}

	// Wait for the rate limiter
	t.rateLimiter[base].Take()

	// Get the schema
	schemaResp := &GetSchemaResponses{}
	resp, err := client.R().
		SetHeader("Authorization", "Bearer "+token).
		SetPathParam("baseId", base).
		SetResult(schemaResp).
		Get(getSchemaURL)

	if err != nil {
		return nil, nil, fmt.Errorf("failed to get schema: %w", err)
	}

	if resp.IsError() {
		return nil, nil, fmt.Errorf("failed to get schema(%d): %s", resp.StatusCode(), resp.String())
	}

	// Find the table
	tableID := ""
	tableIndex := -1
	for i, tableSchema := range schemaResp.Tables {
		if tableSchema.Name == table || tableSchema.ID == table {
			tableID = tableSchema.ID
			tableIndex = i
			break
		}
	}

	if tableID == "" {
		return nil, nil, fmt.Errorf("table %s not found in base %s", table, base)
	}

	schema := []rpc.DatabaseSchemaColumn{
		{
			Name:        "view",
			Type:        rpc.ColumnTypeString,
			IsParameter: true,
			IsRequired:  false,
		},
		{
			Name: "id",
			Type: rpc.ColumnTypeString,
		},
		{
			Name: "created_at",
			Type: rpc.ColumnTypeString,
		},
	}

	// Map a column name with its index and type
	mapColumns := make(map[string]column)

	// Compute the schema
	airtableSchema := schemaResp.Tables[tableIndex]
	for _, field := range airtableSchema.Fields {
		// If the field has no name, use the ID
		name := field.Name
		if name == "" {
			name = field.ID
		}
		switch field.Type {
		case
			"aiText",
			"multipleAttachments",
			"barcode",
			"button",
			"singleCollaborator",
			"createdBy",
			"createdTime",
			"date",
			"dateTime",
			"email",
			"lastModifiedBy",
			"lastModifiedTime",
			"multipleRecordLinks",
			"multilineText",
			"multipleCollaborators",
			"multipleSelects",
			"phoneNumber",
			"richText",
			"singleLineText",
			"singleSelect",
			"externalSyncSource", "url":
			schema = append(schema, rpc.DatabaseSchemaColumn{
				Name: name,
				Type: rpc.ColumnTypeString,
			})
		case
			"autoNumber",
			"checkbox",
			"count",
			"duration",
			"formula",
			"multipleLookupValues",
			"rollup":
			schema = append(schema, rpc.DatabaseSchemaColumn{
				Name: name,
				Type: rpc.ColumnTypeInt,
			})
		case
			"currency",
			"number",
			"percent",
			"rating":
			schema = append(schema, rpc.DatabaseSchemaColumn{
				Name: name,
				Type: rpc.ColumnTypeFloat,
			})
		default:
			log.Printf("unsupported type %s for column %s", field.Type, name)
			continue

		}
		mapColumns[name] = column{
			index:   len(schema) - 2, // Because view is ignored
			name:    name,
			colType: field.Type,
		}
	}

	var db *badger.DB
	if cacheEnabled {
		// Open the cache database
		hashedToken := md5.Sum([]byte(token))
		hashedBase := md5.Sum([]byte(base))
		hashedTable := md5.Sum([]byte(table))

		cacheFolder := path.Join(xdg.CacheHome, "anyquery", "plugins", "airtable",
			fmt.Sprintf("%x", hashedToken[:]),
			fmt.Sprintf("%x", hashedBase[:]),
			fmt.Sprintf("%x", hashedTable[:]))
		err = os.MkdirAll(cacheFolder, 0755)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to create cache folder: %w", err)
		}

		// Open the badger database encrypted with the toke
		options := badger.DefaultOptions(cacheFolder).WithEncryptionKey(hashedToken[:]).
			WithNumVersionsToKeep(1).WithCompactL0OnClose(true).WithValueLogFileSize(2 << 24).
			WithIndexCacheSize(2 << 22)
		badgerDB, err := badger.Open(options)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to open badger database: %w", err)
		}
		db = badgerDB
	}

	return &airtableTable{
			cols:        mapColumns,
			token:       token,
			base:        base,
			table:       table,
			rateLimiter: t.rateLimiter[base],
			cacheDB:     db,
		}, &rpc.DatabaseSchema{
			Columns:       schema,
			HandlesInsert: true,
			HandlesUpdate: true,
			HandlesDelete: true,
			PrimaryKey:    1,
			BufferInsert:  9,
			BufferUpdate:  9,
			BufferDelete:  9,
		}, nil
}

type airtableTable struct {
	cols        map[string]column
	token       string
	base        string
	table       string
	rateLimiter ratelimit.Limiter
	cacheDB     *badger.DB
}

type airtableCursor struct {
	cols        map[string]column
	nextCursor  string
	token       string
	base        string
	table       string
	rateLimiter ratelimit.Limiter
	cacheDB     *badger.DB
}

// Return a slice of rows that will be returned to Anyquery and filtered.
// The second return value is true if the cursor has no more rows to return
//
// The constraints are used for optimization purposes to "pre-filter" the rows
// If the rows returned don't match the constraints, it's not an issue. Anyquery will filter them out
func (t *airtableCursor) Query(constraints rpc.QueryConstraint) ([][]interface{}, bool, error) {
	var view string
	for _, constraint := range constraints.Columns {
		if constraint.ColumnID == 0 {
			rawVal, ok := constraint.Value.(string)
			if ok {
				view = rawVal
			}
		}
	}
	// Try to get the rows from the cache
	cacheKey := fmt.Sprintf("list-%s-%s-%s-%s", t.base, t.table, t.nextCursor, view)
	fromCache := false
	listRecordsResp := &ListRecordsResponse{}

	if t.cacheDB != nil {
		err := t.cacheDB.View(func(txn *badger.Txn) error {
			item, err := txn.Get([]byte(cacheKey))
			if err != nil {
				return err
			}

			err = item.Value(func(val []byte) error {
				decoder := gob.NewDecoder(bytes.NewReader(val))
				err = decoder.Decode(listRecordsResp)
				if err != nil {
					return err
				}
				return nil
			})
			if err != nil {
				return err
			}
			return nil
		})
		fromCache = err == nil
		if err != nil {
			log.Printf("failed to get %s from cache: %s", cacheKey, err)
		}
	}

	// Request the records from the API
	if !fromCache {
		t.rateLimiter.Take()
		resp, err := client.R().
			SetHeader("Authorization", "Bearer "+t.token).
			SetResult(listRecordsResp).
			SetQueryParams(map[string]string{
				"offset":   t.nextCursor,
				"pageSize": "100",
				"view":     view,
			}).
			SetPathParams(map[string]string{
				"baseId":  t.base,
				"tableId": t.table,
			}).
			Get(tableURL)

		if err != nil {
			return nil, false, fmt.Errorf("failed to get records: %w", err)
		}

		if resp.IsError() {
			return nil, false, fmt.Errorf("failed to get records(%d): %s", resp.StatusCode(), resp.String())
		}

		// Save the records in the cache
		if t.cacheDB != nil {
			err := t.cacheDB.Update(func(txn *badger.Txn) error {
				var buf bytes.Buffer
				encoder := gob.NewEncoder(&buf)
				err := encoder.Encode(listRecordsResp)
				if err != nil {
					return err
				}

				e := badger.NewEntry([]byte(cacheKey), buf.Bytes()).WithTTL(time.Hour)
				err = txn.SetEntry(e)
				if err != nil {
					return err
				}
				return nil
			})
			if err != nil {
				log.Printf("failed to save %s in cache: %s", cacheKey, err)
			}
		}
	}

	rows := make([][]interface{}, 0, len(listRecordsResp.Records))
	for _, record := range listRecordsResp.Records {
		row := make([]interface{}, len(t.cols)+2)
		row[0] = record.ID
		row[1] = record.CreatedTime

		for name, value := range record.Fields {
			if col, ok := t.cols[name]; ok {
				// Unmarshal the value
				row[col.index] = unmarshal(value, col.colType)
			}
		}

		rows = append(rows, row)

	}

	t.nextCursor = listRecordsResp.Offset

	return rows, len(rows) < 100 || listRecordsResp.Offset == "", nil
}

// Create a new cursor that will be used to read rows
func (t *airtableTable) CreateReader() rpc.ReaderInterface {
	return &airtableCursor{
		cols:        t.cols,
		token:       t.token,
		base:        t.base,
		table:       t.table,
		rateLimiter: t.rateLimiter,
		cacheDB:     t.cacheDB,
	}
}

// A slice of rows to insert
func (t *airtableTable) Insert(rows [][]interface{}) error {
	req := InsertRecordRequest{}

	for _, row := range rows {
		item := InsertRecordItem{
			Fields: make(map[string]interface{}),
		}
		for name, col := range t.cols {
			// We have to add 1 because the first column is the view
			if col.index+1 < 3 {
				continue
			}
			if col.index+1 >= len(row) {
				continue
			}
			if row[col.index+1] == nil {
				continue
			}

			marshalled := marshal(row[col.index+1], col.colType)
			if marshalled == nil {
				continue
			}
			item.Fields[name] = marshalled

		}
		req.Records = append(req.Records, item)
	}

	t.rateLimiter.Take()
	resp, err := client.R().
		SetHeader("Authorization", "Bearer "+t.token).
		SetHeader("Content-Type", "application/json").
		SetBody(req).
		SetPathParams(map[string]string{
			"baseId":  t.base,
			"tableId": t.table,
		}).
		Post(tableURL)

	if err != nil {
		return fmt.Errorf("failed to insert records: %w", err)
	}

	if resp.IsError() {
		return fmt.Errorf("failed to insert records(%d): %s", resp.StatusCode(), resp.String())
	}

	// Clear the cache
	// to avoid inconsistencies
	t.clearCache()

	return nil
}

// A slice of rows to update
// The first element of each row is the primary key
// while the rest are the values to update
// The primary key is therefore present twice
func (t *airtableTable) Update(rows [][]interface{}) error {
	req := UpdateRecordRequest{}
	for _, row := range rows {
		item := UpdateRecordItem{
			ID:     row[0].(string),
			Fields: make(map[string]interface{}),
		}
		row = row[1:] // Remove the primary key

		for name, col := range t.cols {
			// We have to add 1 because the first column is the view
			if col.index+1 < 3 {
				continue
			}
			if col.index+1 >= len(row) {
				continue
			}
			if row[col.index+1] == nil {
				continue
			}

			marshalled := marshal(row[col.index+1], col.colType)
			if marshalled == nil {
				continue
			}
			item.Fields[name] = marshalled
		}
		req.Records = append(req.Records, item)
	}

	bodyMarshalled, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	t.rateLimiter.Take()

	resp, err := client.R().
		SetHeader("Authorization", "Bearer "+t.token).
		SetHeader("Content-Type", "application/json").
		SetBody(string(bodyMarshalled)).
		SetPathParams(map[string]string{
			"baseId":  t.base,
			"tableId": t.table,
		}).
		Patch(tableURL)

	if err != nil {
		return fmt.Errorf("failed to update records: %w", err)
	}

	if resp.IsError() {
		return fmt.Errorf("failed to update records(%d): %s", resp.StatusCode(), resp.String())
	}

	// Clear the cache
	// to avoid inconsistencies
	t.clearCache()

	return nil
}

// A slice of primary keys to delete
func (t *airtableTable) Delete(primaryKeys []interface{}) error {
	pks := []string{}
	for _, key := range primaryKeys {
		if keyStr, ok := key.(string); ok {
			pks = append(pks, keyStr)
		}
	}

	urlValues := url.Values{}
	for _, pk := range pks {
		urlValues.Add("records[]", pk)
	}
	res, err := client.R().
		SetHeader("Authorization", "Bearer "+t.token).
		SetHeader("Content-Type", "application/json").
		SetPathParams(map[string]string{
			"baseId":  t.base,
			"tableId": t.table,
		}).
		SetQueryParamsFromValues(urlValues).
		Delete(tableURL)

	if err != nil {
		return fmt.Errorf("failed to delete records: %w", err)
	}

	if res.IsError() {
		return fmt.Errorf("failed to delete records(%d): %s", res.StatusCode(), res.String())
	}

	// Clear the cache
	// to avoid inconsistencies
	t.clearCache()

	return nil
}

func (t *airtableTable) clearCache() {
	if t.cacheDB != nil {
		err := t.cacheDB.DropPrefix([]byte("list-"))
		if err != nil {
			log.Printf("failed to clear cache: %s", err)
		}
	}
}

// A destructor to clean up resources
func (t *airtableTable) Close() error {
	if t.cacheDB != nil {
		err := t.cacheDB.Close()
		if err != nil {
			return fmt.Errorf("failed to close cache: %w", err)
		}
	}
	return nil
}
