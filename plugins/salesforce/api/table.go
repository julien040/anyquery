package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/url"
	"slices"
	"strings"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/hashicorp/go-retryablehttp"
	"github.com/huandu/go-sqlbuilder"
	"github.com/julien040/anyquery/rpc"
	"github.com/julien040/anyquery/rpc/helper"
)

func TableFactory(sObjectGlob string) func(args rpc.TableCreatorArgs) (rpc.Table, *rpc.DatabaseSchema, error) {
	return func(args rpc.TableCreatorArgs) (rpc.Table, *rpc.DatabaseSchema, error) {
		// Init a new client
		sObject := sObjectGlob
		domain := strings.TrimRight(args.UserConfig.GetString("domain"), "/ ")
		if domain == "" {
			return nil, nil, fmt.Errorf("domain must be set in the plugin configuration")
		}
		httpClient, err := GetAuthHTTPClient(args.UserConfig)
		if err != nil {
			return nil, nil, err
		}
		retry := retryablehttp.NewClient()
		retry.HTTPClient = httpClient
		retry.RetryMax = 1

		restyClient := resty.NewWithClient(retry.StandardClient())

		// Get the schema
		cols, colIndex, schema, err := InspectSchema(restyClient, sObject, domain)
		if err != nil {
			return nil, nil, fmt.Errorf("unable to inspect schema: %w", err)
		}

		// Open the cache
		cache, err := helper.NewCache(helper.NewCacheArgs{
			Paths:         []string{"salesforce", domain, sObject},
			EncryptionKey: []byte(args.UserConfig.GetString("encryption_key")),
			MaxMemSize:    50 * 1024 * 1024,
			MaxSize:       250 * 1024 * 1024,
		})

		cacheTTL := args.UserConfig.GetInt("cache_ttl")
		if cacheTTL == 0 {
			// Try to convert the float to an int
			cacheTTL = int64(args.UserConfig.GetFloat("cache_ttl"))
		}

		// To ensure the user didn't set a negative value
		if cacheTTL < 0 {
			cacheTTL = 0
		}

		if err != nil {
			log.Printf("unable to open cache: %v", err)
		}

		return &TableSalesforce{
				colMapper:       cols,
				colIndex:        colIndex,
				sObject:         sObject,
				domain:          domain,
				restyClient:     restyClient,
				allOrNone:       args.UserConfig.GetBool("allOrNone"),
				cache:           cache,
				secondsCacheTTL: cacheTTL,
			}, &rpc.DatabaseSchema{
				HandlesInsert: true,
				HandlesUpdate: true,
				HandlesDelete: true,
				PrimaryKey:    0,
				BufferInsert:  200,
				BufferUpdate:  200,
				BufferDelete:  200,
				Columns:       schema,
			}, nil
	}
}

// The table struct
// There is one per connection to the plugin and is created by the creator function
// In there, you can store any state you need to read the rows (e.g. a database connection, an API token, etc.)
type TableSalesforce struct {
	colMapper       ColMapper     // Map the column name to the column information
	colIndex        ColIndex      // Map the column index to the column name
	sObject         string        // The Salesforce object to query
	domain          string        // The domain of the Salesforce instance (e.g. mydomain.my.salesforce.com)
	restyClient     *resty.Client // An HTTP client to interact with Salesforce
	allOrNone       bool          // If true, any failure in a batch will cause the entire batch to fail
	cache           *helper.Cache // Cache for a query
	secondsCacheTTL int64         // How many seconds to cache the records
}

// The cursor struct
// There is one per query and is created by the createReader function
// In there, you can store any state you need to read the rows (e.g. a database connection from accountTable, an offset, a cursor, etc.)
type CursorSalesforce struct {
	colMapper       ColMapper
	colIndex        ColIndex
	sObject         string
	domain          string
	nextRecordsUrl  string
	restyClient     *resty.Client
	allOrNone       bool
	cache           *helper.Cache
	secondsCacheTTL int64
}

// Create a new cursor that will be used to read rows
func (t *TableSalesforce) CreateReader() rpc.ReaderInterface {
	return &CursorSalesforce{
		colMapper:       t.colMapper,
		colIndex:        t.colIndex,
		sObject:         t.sObject,
		domain:          t.domain,
		restyClient:     t.restyClient,
		allOrNone:       t.allOrNone,
		cache:           t.cache,
		secondsCacheTTL: t.secondsCacheTTL,
	}
}

// Return a slice of rows that will be returned to Anyquery and filtered.
// The second return value is true if the cursor has no more rows to return
//
// The constraints are used for optimization purposes to "pre-filter" the rows
// If the rows returned don't match the constraints, it's not an issue. Anyquery will filter them out
func (t *CursorSalesforce) Query(constraints rpc.QueryConstraint) ([][]interface{}, bool, error) {
	// If t.nextRecordsUrl is empty, we need to get the first page
	// and therefore construct the SQL query
	var urlToQuery string
	if t.nextRecordsUrl == "" {
		colNames := make([]string, 0, len(t.colMapper))
		for colName := range t.colMapper {
			colNames = append(colNames, colName)
		}

		// Order the colNames
		// This is needed to have a consistent cache key
		slices.Sort(colNames)

		// Create the SQL query
		sqlBuilder := sqlbuilder.NewSelectBuilder()

		sqlBuilder = sqlbuilder.Select(colNames...).From(t.sObject)
		whereExpr := []string{}
		// Add the where clause
		for _, constraint := range constraints.Columns {
			// Retrieve the column name from the column index
			if colName, ok := t.colIndex[constraint.ColumnID]; ok {
				// Ensure the column is filterable
				if val, ok := t.colMapper[colName]; !ok || !val.SalesforceFilterable {
					continue
				}
				// Add the where clause
				var expr string
				switch constraint.Operator {
				case rpc.OperatorEqual:
					expr = sqlBuilder.Equal(colName, constraint.Value)
				case rpc.OperatorNotEqual:
					expr = sqlBuilder.NotEqual(colName, constraint.Value)
				case rpc.OperatorGreater:
					expr = sqlBuilder.GreaterThan(colName, constraint.Value)
				case rpc.OperatorGreaterOrEqual:
					expr = sqlBuilder.GreaterEqualThan(colName, constraint.Value)
				case rpc.OperatorLess:
					expr = sqlBuilder.LessThan(colName, constraint.Value)
				case rpc.OperatorLessOrEqual:
					expr = sqlBuilder.LessEqualThan(colName, constraint.Value)
				case rpc.OperatorGlob:
					// Replace the * with % for the LIKE clause
					value := strings.ReplaceAll(constraint.Value.(string), "*", "%")
					expr = sqlBuilder.Like(colName, value)
				}
				if expr != "" {
					whereExpr = append(whereExpr, expr)
				}
			}

		}
		if len(whereExpr) > 0 {
			sqlBuilder = sqlBuilder.Where(whereExpr...)
		}

		// Compute the URL
		queryPlaceholder, args := sqlBuilder.Build()
		query, err := sqlbuilder.MySQL.Interpolate(queryPlaceholder, args)
		if err != nil {
			return nil, true, fmt.Errorf("unable to interpolate query: %w", err)
		}
		urlToQuery = fmt.Sprintf("https://%s/services/data/v61.0/query?q=%s", t.domain, url.QueryEscape(query))
	} else {
		urlToQuery = fmt.Sprintf("https://%s%s", t.domain, t.nextRecordsUrl)
	}

	// Get the records from cache if possible, otherwise fetch them
	var records [][]interface{}
	if t.cache != nil {
		records, metadata, err := t.cache.Get(urlToQuery)
		log.Printf("cache get %s (%d): %v ", urlToQuery, len(records), err)
		if len(records) > 0 && err == nil {
			log.Printf("got records from cache")
			done := true
			if val, ok := metadata["nextRecordsUrl"]; ok {
				t.nextRecordsUrl = val.(string)
				done = t.nextRecordsUrl == ""
			}
			return records, done, nil
		}
	}
	log.Printf("fetching records from Salesforce")
	bodyResp := &Rows{}
	resp, err := t.restyClient.R().SetResult(bodyResp).Get(urlToQuery)
	if err != nil {
		return nil, true, fmt.Errorf("unable to get records: %w", err)
	}
	if resp.IsError() {
		return nil, true, fmt.Errorf("unable to get records(%d): %s", resp.StatusCode(), resp.String())
	}

	// Update the nextRecordsUrl
	t.nextRecordsUrl = bodyResp.NextRecordsURL

	// Map the records
	records = make([][]interface{}, 0, len(bodyResp.Records))
	for _, record := range bodyResp.Records {
		row := make([]interface{}, len(t.colMapper))
		row[0] = record["Id"]
		for colName, col := range t.colMapper {
			if col.Index >= len(row) || col.Index <= 0 {
				// Skip invalid index
				// and the Id field
				continue
			}
			if value, ok := record[colName]; ok {
				row[col.Index] = helper.Serialize(value)
			}
		}
		records = append(records, row)
	}

	// Cache the records
	if t.cache != nil {
		metadata := map[string]interface{}{
			"nextRecordsUrl": t.nextRecordsUrl,
		}
		err = t.cache.Set(urlToQuery, records, metadata, time.Duration(t.secondsCacheTTL)*time.Second)
		log.Printf("cache set %s (%d): %v for %d seconds", urlToQuery, len(records), err, t.secondsCacheTTL)
		if err != nil {
			log.Printf("unable to cache records: %v", err)
		}
	}

	return records, t.nextRecordsUrl == "", nil
}

// A slice of rows to insert
func (t *TableSalesforce) Insert(rows [][]interface{}) error {
	requestBody := InsertUpdateRequest{
		Records:   make([]InsertObject, 0, len(rows)),
		AllOrNone: t.allOrNone,
	}

	for _, row := range rows {
		record := make(InsertObject)
		for colName, col := range t.colMapper {
			if col.Index >= len(row) || col.Index <= 0 {
				// Skip invalid index
				continue
			}

			// If the value is nil, skip it
			if row[col.Index] == nil {
				continue
			}

			// If the column is not updateable, skip it
			if !col.SalesforceUpdateable {
				continue
			}

			// If the value is a string, we try to parse it as JSON
			// If it fails, we keep the string
			// This is needed for some fields, such as billingAddress
			if str, ok := row[col.Index].(string); ok {
				var value interface{}
				err := json.Unmarshal([]byte(str), &value)
				if err != nil {
					value = str
				}
				record[colName] = value
			} else {
				record[colName] = row[col.Index]
			}
		}

		// Add the object type
		record["attributes"] = InsertUpdateObjectType{
			Type: t.sObject,
		}

		// Append the record if it has at least one field to update
		if len(record) > 1 {
			requestBody.Records = append(requestBody.Records, record)
		}
	}

	response := InsertUpdateResponses{}

	// Send the request
	resp, err := t.restyClient.R().
		SetBody(requestBody).SetResult(&response).
		Post(fmt.Sprintf("https://%s/services/data/v61.0/composite/sobjects", t.domain))
	if err != nil {
		return fmt.Errorf("unable to insert records: %w", err)
	}

	if resp.IsError() {
		return fmt.Errorf("unable to insert records(%d): %s", resp.StatusCode(), resp.String())
	}

	// Check for errors
	err = nil
	for i, resp := range response {
		if !resp.Success {
			err = errors.Join(err, fmt.Errorf("unable to insert row %d: %v", i, resp.Errors))
		}
	}

	// Clear the cache if the insert was successful
	if err == nil {
		t.ClearCache()
	}

	return err
}

func (t *TableSalesforce) ClearCache() {
	if t.cache != nil {
		err := t.cache.Clear()
		if err != nil {
			log.Printf("unable to clear cache: %v", err)
		}
	}
}

// A slice of rows to update
// The first element of each row is the primary key
// while the rest are the values to update
// The primary key is therefore present twice
func (t *TableSalesforce) Update(rows [][]interface{}) error {

	requestBody := InsertUpdateRequest{
		Records:   make([]InsertObject, 0, len(rows)),
		AllOrNone: t.allOrNone,
	}

	for _, row := range rows {
		record := make(InsertObject)

		// Add the ID field
		record["id"] = row[0]

		// Add the object type
		record["attributes"] = InsertUpdateObjectType{
			Type: t.sObject,
		}

		// Add the fields
		// Note: Because the first element is the primary key, we start at 1
		// and therefore add +1 everywhere we want to access the row
		for colName, col := range t.colMapper {
			if col.Index+1 >= len(row) || col.Index+1 <= 0 {
				// Skip invalid index
				continue
			}

			// If the value is nil, skip it
			if row[col.Index+1] == nil {
				continue
			}

			// If the column is not updateable, skip it
			if !col.SalesforceUpdateable {
				continue
			}

			// If the value is a string, we try to parse it as JSON
			// If it fails, we keep the string
			// This is needed for some fields, such as billingAddress
			if str, ok := row[col.Index+1].(string); ok {
				var value interface{}
				err := json.Unmarshal([]byte(str), &value)
				if err != nil {
					value = str
				}
				record[colName] = value
			} else {
				record[colName] = row[col.Index+1]
			}

		}

		// Append the record if it has at least one field to update
		if len(record) > 2 {
			requestBody.Records = append(requestBody.Records, record)
		}
	}

	response := InsertUpdateResponses{}
	resp, err := t.restyClient.R().
		SetBody(requestBody).SetResult(&response).
		Patch(fmt.Sprintf("https://%s/services/data/v61.0/composite/sobjects", t.domain))

	if err != nil {
		return fmt.Errorf("unable to update records: %w", err)
	}

	if resp.IsError() {
		return fmt.Errorf("unable to update records(%d): %s", resp.StatusCode(), resp.String())
	}

	// Check for errors
	err = nil
	for i, resp := range response {
		if !resp.Success {
			err = errors.Join(err, fmt.Errorf("unable to update row %d: %v", i, resp.Errors))
		}
	}

	// Clear the cache if the insert was successful
	if err == nil {
		t.ClearCache()
	}

	return err
}

// A slice of primary keys to delete
func (t *TableSalesforce) Delete(primaryKeys []interface{}) error {
	url := fmt.Sprintf("https://%s/services/data/v61.0/composite/sobjects", t.domain)

	ids := strings.Builder{}
	for i, id := range primaryKeys {
		if i > 0 {
			ids.WriteString(",")
		}
		if strVal, ok := id.(string); ok {
			ids.WriteString(strVal)
		}
	}
	allOrNone := "false"
	if t.allOrNone {
		allOrNone = "true"
	}

	response := InsertUpdateResponses{}

	resp, err := t.restyClient.R().
		SetQueryParam("ids", ids.String()).
		SetQueryParam("allOrNone", allOrNone).
		SetResult(&response).
		Delete(url)

	if err != nil {
		return fmt.Errorf("unable to delete records: %w", err)
	}

	if resp.IsError() {
		return fmt.Errorf("unable to delete records(%d): %s", resp.StatusCode(), resp.String())
	}

	err = nil
	for i, resp := range response {
		if !resp.Success {
			err = errors.Join(err, fmt.Errorf("unable to update row %d: %v", i, resp.Errors))
		}
	}

	// Clear the cache if the insert was successful
	if err == nil {
		t.ClearCache()
	}

	return err
}

// A destructor to clean up resources
func (t *TableSalesforce) Close() error {
	// Close the cache
	if t.cache != nil {
		err := t.cache.Close()
		if err != nil {
			return fmt.Errorf("unable to close cache: %w", err)
		}
	}
	return nil
}
