package main

import (
	"crypto/md5"
	"crypto/sha256"
	"fmt"
	"log"
	"strings"
	"time"
	"unicode"

	"github.com/go-resty/resty/v2"
	"github.com/hashicorp/go-retryablehttp"
	"github.com/julien040/anyquery/rpc"
	"github.com/julien040/anyquery/rpc/helper"
)

func alphanumeriseString(s string) string {
	builder := strings.Builder{}
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z':
			builder.WriteRune(r)
		case r >= 'A' && r <= 'Z':
			builder.WriteRune(unicode.ToLower(r))
		case r >= '0' && r <= '9':
			builder.WriteRune(r)
		case r == ' ' || r == '_' || r == '-':
			builder.WriteRune('_')
		}
	}

	return builder.String()
}

type column struct {
	name     string
	index    int
	colType  string
	readOnly bool
}

var retry = retryablehttp.NewClient()
var client = resty.NewWithClient(retry.StandardClient())

var propertyEndpoint = "https://api.hubapi.com/crm/v3/properties/{objectType}?archived=false"
var objectListEndpoint = "https://api.hubapi.com/crm/v3/objects/{objectType}?archived=false"
var objectCreateEndpoint = "https://api.hubapi.com/crm/v3/objects/{objectType}/batch/create"
var objectUpdateEndpoint = "https://api.hubapi.com/crm/v3/objects/{objectType}/batch/update"
var objectDeleteEndpoint = "https://api.hubapi.com/crm/v3/objects/{objectType}/batch/archive"

func factory(objectName string, readOnly bool) rpc.TableCreator {
	return func(args rpc.TableCreatorArgs) (rpc.Table, *rpc.DatabaseSchema, error) {
		objectName := objectName
		readOnly := readOnly
		token := args.UserConfig.GetString("token")
		if token == "" {
			return nil, nil, fmt.Errorf("token must be set in the plugin configuration")
		}

		hashedToken := sha256.Sum256([]byte(token))
		hashedTokenMD5 := md5.Sum([]byte(token))

		cache, err := helper.NewCache(helper.NewCacheArgs{
			Paths:         []string{"hubspot", fmt.Sprintf("%x", hashedTokenMD5[:]), objectName},
			EncryptionKey: hashedToken[:],
		})

		if err != nil {
			return nil, nil, fmt.Errorf("could not open cache: %w", err)
		}

		// Get the properties from the API
		properties := Properties{}
		resp, err := client.R().SetHeader("Authorization", "Bearer "+token).SetResult(&properties).
			SetPathParam("objectType", objectName).Get(propertyEndpoint)

		if err != nil {
			return nil, nil, fmt.Errorf("failed to get schema of object %s: %w", objectName, err)
		}

		if resp.IsError() {
			return nil, nil, fmt.Errorf("failed to get schema of object %s(%d): %s", objectName, resp.StatusCode(), resp.String())
		}

		schema := []rpc.DatabaseSchemaColumn{}

		// Map a field name to a column
		mapColName := map[string]column{}

		// Map a column index to a field name
		mapIndexName := map[int]string{}

		// Ensure that there are no duplicate column names
		duplicateCheck := map[string]struct{}{}

		j := 0
	mainLoop:
		for _, property := range properties.Results {
			colName := alphanumeriseString(property.Label)
			if _, ok := duplicateCheck[colName]; ok {
				continue mainLoop
			}
			switch property.Type {
			case "enumeration", "string":
				schema = append(schema, rpc.DatabaseSchemaColumn{
					Name:        colName,
					Type:        rpc.ColumnTypeString,
					Description: property.Description,
				})
			case "datetime", "date":
				schema = append(schema, rpc.DatabaseSchemaColumn{
					Name:        colName,
					Type:        rpc.ColumnTypeDateTime,
					Description: property.Description,
				})
			case "number":
				schema = append(schema, rpc.DatabaseSchemaColumn{
					Name:        colName,
					Type:        rpc.ColumnTypeFloat,
					Description: property.Description,
				})
			case "bool":
				schema = append(schema, rpc.DatabaseSchemaColumn{
					Name:        colName,
					Type:        rpc.ColumnTypeBool,
					Description: property.Description,
				})
			default:
				continue mainLoop
			}
			duplicateCheck[colName] = struct{}{}
			mapIndexName[j] = property.Name

			mapColName[property.Name] = column{
				name:     colName,
				index:    j,
				colType:  property.Type,
				readOnly: property.ModificationMetadata.ReadOnlyValue,
			}
			j++
		}

		// Append created_at and updated_at
		schema = append(schema, rpc.DatabaseSchemaColumn{
			Name:        "record_created_at",
			Type:        rpc.ColumnTypeDateTime,
			Description: "The date and time the record was created in RFC3339 format",
		})
		schema = append(schema, rpc.DatabaseSchemaColumn{
			Name:        "record_updated_at",
			Type:        rpc.ColumnTypeDateTime,
			Description: "The date and time the record was last updated in RFC3339 format",
		})

		// The field name for the primary key is hs_object_id
		pk := -1
		if col, ok := mapColName["hs_object_id"]; ok {
			pk = col.index
		}

		// Get the cache TTL from the user configuration
		cacheTTL := int64(0)
		if args.UserConfig.GetInt("cache_ttl") > 0 {
			cacheTTL = args.UserConfig.GetInt("cache_ttl")
		} else if args.UserConfig.GetFloat("cache_ttl") > 0 {
			cacheTTL = int64(args.UserConfig.GetFloat("cache_ttl"))
		}

		return &hubspotTable{
				token:        token,
				objectType:   objectName,
				cache:        cache,
				mapColName:   mapColName,
				mapIndexName: mapIndexName,
				cacheTTL:     cacheTTL,
			}, &rpc.DatabaseSchema{
				HandlesInsert: !readOnly,
				HandlesUpdate: pk != -1 && !readOnly, // If there is no primary key, we can't update
				HandlesDelete: pk != -1 && !readOnly, // If there is no primary key, we can't delete
				HandleOffset:  false,
				PrimaryKey:    pk,
				Columns:       schema,
				BufferInsert:  100,
				BufferUpdate:  100,
				BufferDelete:  100,
			}, nil
	}
}

// The table struct
// There is one per connection to the plugin and is created by the creator function
// In there, you can store any state you need to read the rows (e.g. a database connection, an API token, etc.)
type hubspotTable struct {
	token        string
	objectType   string
	cache        *helper.Cache
	mapColName   map[string]column
	mapIndexName map[int]string
	cacheTTL     int64
}

// The cursor struct
// There is one per query and is created by the createReader function
// In there, you can store any state you need to read the rows (e.g. a database connection from hubspotTable, an offset, a cursor, etc.)
type hubspotCursor struct {
	token        string
	objectType   string
	cache        *helper.Cache
	mapColName   map[string]column
	mapIndexName map[int]string
	after        string
	cacheTTL     int64
}

// Create a new cursor that will be used to read rows
func (t *hubspotTable) CreateReader() rpc.ReaderInterface {
	return &hubspotCursor{
		token:        t.token,
		objectType:   t.objectType,
		cache:        t.cache,
		mapColName:   t.mapColName,
		mapIndexName: t.mapIndexName,
		after:        "",
		cacheTTL:     t.cacheTTL,
	}
}

// Return a slice of rows that will be returned to Anyquery and filtered.
// The second return value is true if the cursor has no more rows to return
//
// The constraints are used for optimization purposes to "pre-filter" the rows
// If the rows returned don't match the constraints, it's not an issue. Anyquery will filter them out
func (t *hubspotCursor) Query(constraints rpc.QueryConstraint) ([][]interface{}, bool, error) {
	var rows [][]interface{}
	var metadata map[string]interface{}

	cacheKey := fmt.Sprintf("%s_%s", t.objectType, t.after)

	// Try to get the rows from the cache
	rows, metadata, err := t.cache.Get(cacheKey)
	if err == nil {
		t.after = metadata["after"].(string)
		return rows, metadata["after"] == "", nil
	}

	response := Objects{}
	req := client.R().
		SetHeader("Authorization", "Bearer "+t.token).
		SetResult(&response).
		SetPathParam("objectType", t.objectType).
		SetQueryParam("limit", "100")

	if t.after != "" {
		req.SetQueryParam("after", t.after)
	}

	resp, err := req.Get(objectListEndpoint)
	if err != nil {
		return nil, true, fmt.Errorf("failed to get objects: %w", err)
	}

	if resp.IsError() {
		return nil, true, fmt.Errorf("failed to get objects(%d): %s", resp.StatusCode(), resp.String())
	}

	rows = make([][]interface{}, 0, len(response.Results))
	for _, object := range response.Results {
		row := make([]interface{}, len(t.mapColName)+2)
		for key, value := range object.Properties {
			if col, ok := t.mapColName[key]; ok {
				row[col.index] = helper.Serialize(value)
			}
		}
		row = append(row, object.CreatedAt)
		row = append(row, object.UpdatedAt)
		rows = append(rows, row)
	}

	t.after = response.Paging.Next.After
	metadata = map[string]interface{}{
		"after": t.after,
	}

	// Save the rows in the cache
	err = t.cache.Set(cacheKey, rows, metadata, time.Duration(t.cacheTTL)*time.Second)
	if err != nil {
		log.Printf("failed to save cache: %v", err)
	}

	return rows, t.after == "", nil
}

// A slice of rows to insert
func (t *hubspotTable) Insert(rows [][]interface{}) error {
	// Example: insert the rows in a database
	// for _, row := range rows {
	// 	err := db.Insert(row[0], row[1], row[2])
	// 	if err != nil {
	// 		return err
	// 	}

	body := CreateUpdateBody{
		Inputs: make([]InputUpdate, 0, len(rows)),
	}

	for _, row := range rows {
		properties := map[string]interface{}{}
		for i, value := range row[:len(row)-2] {
			if _, ok := t.mapIndexName[i]; !ok {
				continue
			}
			colInfo := t.mapColName[t.mapIndexName[i]]
			if !colInfo.readOnly {
				properties[t.mapIndexName[i]] = value
			}
		}

		body.Inputs = append(body.Inputs, InputUpdate{
			Properties: properties,
		})
	}

	resp, err := client.R().
		SetHeader("Authorization", "Bearer "+t.token).
		SetBody(body).
		SetPathParam("objectType", t.objectType).
		Post(objectCreateEndpoint)

	if err != nil {
		return fmt.Errorf("failed to insert objects: %w", err)
	}

	if resp.IsError() {
		return fmt.Errorf("failed to insert objects(%d): %s", resp.StatusCode(), resp.String())
	}

	// Clear the cache
	t.cache.Clear()

	return nil
}

// A slice of rows to update
// The first element of each row is the primary key
// while the rest are the values to update
// The primary key is therefore present twice
func (t *hubspotTable) Update(rows [][]interface{}) error {
	body := CreateUpdateBody{
		Inputs: make([]InputUpdate, 0, len(rows)),
	}

	for _, row := range rows {
		properties := map[string]interface{}{}
		for i, value := range row[1 : len(row)-2] {
			// If the property does not exist, we skip it
			if _, ok := t.mapIndexName[i]; !ok {
				continue
			}

			colInfo := t.mapColName[t.mapIndexName[i]]

			// If the property is read-only, we skip it
			if colInfo.readOnly {
				continue
			}

			properties[t.mapIndexName[i]] = value

		}

		pk := row[0]
		// Sometimes, SQLite might transform the primary key to a float
		switch val := pk.(type) {
		case float64, int, int64:
			pk = fmt.Sprintf("%d", val)
		case string:
			pk = val
		default:
			log.Printf("Unknown type for pk in update: %T", pk)
		}

		body.Inputs = append(body.Inputs, InputUpdate{
			ID:         row[0].(string),
			Properties: properties,
		})

	}

	resp, err := client.R().
		SetHeader("Authorization", "Bearer "+t.token).
		SetBody(body).
		SetPathParam("objectType", t.objectType).
		Post(objectUpdateEndpoint)

	if err != nil {
		return fmt.Errorf("failed to update objects: %w", err)
	}

	if resp.IsError() {
		return fmt.Errorf("failed to update objects(%d): %s", resp.StatusCode(), resp.String())
	}

	// Clear the cache
	t.cache.Clear()

	return nil
}

// A slice of primary keys to delete
func (t *hubspotTable) Delete(primaryKeys []interface{}) error {
	body := CreateUpdateBody{
		Inputs: make([]InputUpdate, 0, len(primaryKeys)),
	}

	for _, pk := range primaryKeys {
		// Sometimes, SQLite might transform the primary key to a float
		// We need to convert it back to a string
		pkStr := ""
		switch val := pk.(type) {
		case string:
			pkStr = val
		case float64, int, int64:
			pkStr = fmt.Sprintf("%d", val)
		default:
			log.Printf("Unknown type for pk in delete: %T", pk)
		}

		body.Inputs = append(body.Inputs, InputUpdate{
			ID: pkStr,
		})
	}

	resp, err := client.R().
		SetHeader("Authorization", "Bearer "+t.token).
		SetBody(body).
		SetPathParam("objectType", t.objectType).
		Post(objectDeleteEndpoint)

	if err != nil {
		return fmt.Errorf("failed to delete objects: %w", err)
	}

	if resp.IsError() {
		return fmt.Errorf("failed to delete objects(%d): %s", resp.StatusCode(), resp.String())
	}

	// Clear the cache
	t.cache.Clear()

	return nil
}

// A destructor to clean up resources
func (t *hubspotTable) Close() error {
	t.cache.Close()
	return nil
}
