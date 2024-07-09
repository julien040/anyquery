package main

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"sort"
	"strings"
	"time"

	"github.com/adrg/xdg"
	"github.com/dgraph-io/badger/v4"
	"github.com/hashicorp/go-retryablehttp"
	"github.com/jomei/notionapi"
	"github.com/julien040/anyquery/rpc"
)

func databaseTable(args rpc.TableCreatorArgs) (rpc.Table, *rpc.DatabaseSchema, error) {
	// Check if the user config contains a token
	token, ok := args.UserConfig["token"]
	if !ok {
		return nil, nil, fmt.Errorf("missing token in user config")
	}
	tokenStr, ok := token.(string)
	if !ok {
		return nil, nil, fmt.Errorf("token must be a string")
	}

	// Create a Notion client that uses hashicorp/retryablehttp to retry requests
	// It's nice because it reads the Retry-After header and waits for the specified time
	httpClient := retryablehttp.NewClient()

	client := notionapi.NewClient(notionapi.Token(tokenStr), notionapi.WithHTTPClient(httpClient.StandardClient()))

	// Check if the user config contains a database id
	databaseID, ok := args.UserConfig["database_id"]
	if !ok {
		return nil, nil, fmt.Errorf("missing database_id in user config")
	}
	databaseIDStr, ok := databaseID.(string)
	if !ok {
		return nil, nil, fmt.Errorf("database_id must be a string")
	}

	// Retrieve the databaseSchema schema
	databaseSchema, err := client.Database.Get(context.Background(), notionapi.DatabaseID(databaseIDStr))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get database. Ensure that the database_id is correct."+
			"(e.g. in https://www.notion.so/myworkspace/a8aec43384f447ed84390e8e42c2e089?v=…….,"+
			" a8aec43384f447ed84390e8e42c2e089 is the database_id): %w", err)
	}

	// These are columns that are always present for every page
	// We prefix them with an underscore to avoid conflicts with user defined columns
	// If a user defined column has the same name as one of these, the database won't work
	rpcColumns := make([]rpc.DatabaseSchemaColumn, 6, len(databaseSchema.Properties)+6)
	rpcColumns[0] = rpc.DatabaseSchemaColumn{
		Name: "_page_id",
		Type: rpc.ColumnTypeString,
	}
	rpcColumns[1] = rpc.DatabaseSchemaColumn{
		Name: "_created_time",
		Type: rpc.ColumnTypeString,
	}
	rpcColumns[2] = rpc.DatabaseSchemaColumn{
		Name: "_last_edited_time",
		Type: rpc.ColumnTypeString,
	}
	rpcColumns[3] = rpc.DatabaseSchemaColumn{
		Name: "_cover_url",
		Type: rpc.ColumnTypeString,
	}
	rpcColumns[4] = rpc.DatabaseSchemaColumn{
		Name: "_icon_url",
		Type: rpc.ColumnTypeString,
	}
	rpcColumns[5] = rpc.DatabaseSchemaColumn{
		Name: "_page_url",
		Type: rpc.ColumnTypeString,
	}

	notionTableCols := []string{"_page_id", "_created_time", "_last_edited_time", "_cover_url", "_icon_url", "_page_url"}

	for k, prop := range databaseSchema.Properties {
		if k == "_page_id" || k == "_created_time" || k == "_last_edited_time" ||
			k == "_cover_url" || k == "_icon_url" || k == "_page_url" {
			return nil, nil, fmt.Errorf("column name %s is reserved", k)
		}
		switch prop.GetType() {
		case notionapi.PropertyConfigCreatedBy,
			notionapi.PropertyConfigCreatedTime,
			notionapi.PropertyConfigLastEditedBy,
			notionapi.PropertyConfigLastEditedTime,
			notionapi.PropertyConfigTypeDate,
			notionapi.PropertyConfigStatus,
			notionapi.PropertyConfigTypeEmail,
			notionapi.PropertyConfigTypeFiles,
			notionapi.PropertyConfigTypeMultiSelect,
			notionapi.PropertyConfigTypePeople,
			notionapi.PropertyConfigTypePhoneNumber,
			notionapi.PropertyConfigTypeRichText,
			notionapi.PropertyConfigTypeSelect,
			notionapi.PropertyConfigTypeTitle,
			notionapi.PropertyConfigTypeURL,
			notionapi.PropertyConfigTypeRelation,
			notionapi.PropertyConfigTypeFormula,
			// notionapi.PropertyConfigTypeRollup, // We don't support rollups
			notionapi.PropertyConfigUniqueID: // For some reasons, the UniqueID is never matched because getType returns "". TODO: Fix this
			notionTableCols = append(notionTableCols, k)
			rpcColumns = append(rpcColumns, rpc.DatabaseSchemaColumn{
				Name: k,
				Type: rpc.ColumnTypeString,
			})
		case notionapi.PropertyConfigTypeCheckbox:
			notionTableCols = append(notionTableCols, k)
			rpcColumns = append(rpcColumns, rpc.DatabaseSchemaColumn{
				Name: k,
				Type: rpc.ColumnTypeInt,
			})
		case notionapi.PropertyConfigTypeNumber:
			notionTableCols = append(notionTableCols, k)
			rpcColumns = append(rpcColumns, rpc.DatabaseSchemaColumn{
				Name: k,
				Type: rpc.ColumnTypeFloat,
			})
		default:
			// We don't support this property type
			continue
		}
	}

	// Sort the columns because the map order is not guaranteed
	// If we cache a page, subsequent queries may have columns with the wrong values
	sort.SliceStable(rpcColumns, func(i, j int) bool {
		return rpcColumns[i].Name < rpcColumns[j].Name
	})
	sort.SliceStable(notionTableCols, func(i, j int) bool {
		return notionTableCols[i] < notionTableCols[j]
	})

	// Note the position of _page_id
	// so that we can set it as the primary key
	pkIndex := 0
	for i, col := range rpcColumns {
		if col.Name == "_page_id" {
			pkIndex = i
			break
		}
	}

	schema := rpc.DatabaseSchema{
		HandlesInsert: true,
		HandlesUpdate: true,
		HandlesDelete: true,
		Columns:       rpcColumns,
		HandleOffset:  false,
		PrimaryKey:    pkIndex,
	}

	// Create a cache folder for the plugin
	md5sumToken := md5.Sum([]byte(tokenStr))
	hashedToken := hex.EncodeToString(md5sumToken[:])

	md5sumdb := md5.Sum([]byte(databaseIDStr))
	hashedDatabaseID := hex.EncodeToString(md5sumdb[:])

	cacheFolder := path.Join(xdg.CacheHome, "anyquery", "plugins", "notion", hashedToken, hashedDatabaseID)
	err = os.MkdirAll(cacheFolder, 0755)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create cache folder: %w", err)
	}

	// Open the badger database encrypted with the toke
	options := badger.DefaultOptions(cacheFolder).WithEncryptionKey(md5sumToken[:]).
		WithNumVersionsToKeep(1).WithCompactL0OnClose(true).WithValueLogFileSize(2 << 24).
		WithIndexCacheSize(2 << 24)
	db, err := badger.Open(options)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to open badger database: %w", err)
	}

	return &table{client, databaseSchema, client.Database, notionTableCols, db}, &schema, nil
}

type table struct {
	client   *notionapi.Client
	database *notionapi.Database
	dbClient notionapi.DatabaseService
	columns  []string
	cacheDB  *badger.DB
}

type tableCursor struct {
	nextCursor   string
	database     notionapi.DatabaseService
	databaseInfo *notionapi.Database
	columns      []string
	page         uint64
	cacheDB      *badger.DB
}

// A key to represent a page in the cache
//
// Filter has to be sorted to ensure that the key is the same for the same filter
// Sort also has to be sorted.
type pageCacheKey struct {
	PageID     uint64
	Filter     []rpc.ColumnConstraint
	Sort       []rpc.OrderConstraint
	Properties map[string]notionapi.PropertyConfig
}

type pageCacheValue struct {
	Rows       [][]interface{}
	NextCursor string
	HasMore    bool
}

// GetJSON returns the JSON representation of the key
// This is used to store the key in the cache as bytes
// The function will reorder the filter and sort slices
func (p pageCacheKey) GetJSON() ([]byte, error) {
	// Sort the filter and sort slices
	sort.SliceStable(p.Filter, func(i, j int) bool {
		if p.Filter[i].ColumnID == p.Filter[j].ColumnID {
			return p.Filter[i].Operator < p.Filter[j].Operator
		} else {
			return p.Filter[i].ColumnID < p.Filter[j].ColumnID
		}
	})

	sort.SliceStable(p.Sort, func(i, j int) bool {
		return p.Sort[i].ColumnID < p.Sort[j].ColumnID
	})

	// Marshal the key to JSON
	return json.Marshal(p)
}

func (p pageCacheKey) GetKey() ([]byte, error) {
	key, err := p.GetJSON()
	hash := md5.Sum(key)
	return hash[:], err
}

func (t *tableCursor) Query(constraints rpc.QueryConstraint) ([][]interface{}, bool, error) {
	cacheAvailable := false

	// Try to fetch the page from the cache
	key := pageCacheKey{
		PageID:     t.page,
		Filter:     constraints.Columns,
		Sort:       constraints.OrderBy,
		Properties: t.databaseInfo.Properties,
	}

	var rawValue []byte
	var err error
	var byteCacheKey []byte
	err = t.cacheDB.View(func(txn *badger.Txn) error {
		byteCacheKey, err = key.GetKey()
		if err != nil {
			return err
		}
		item, err := txn.Get(byteCacheKey)
		if err != nil {
			// If the page is not in the cache, we will fetch it from the database
			if err == badger.ErrKeyNotFound {
				return nil
			} else {
				return err
			}
		}
		rawValue, err = item.ValueCopy(nil)
		if err != nil {
			return err
		}
		cacheAvailable = true
		return nil
	})
	if err != nil {
		return nil, false, fmt.Errorf("failed to get page from cache: %w", err)
	}

	if cacheAvailable {
		var value pageCacheValue
		err := json.Unmarshal(rawValue, &value)
		if err == nil {
			t.nextCursor = value.NextCursor
			t.page++
			return value.Rows, !value.HasMore, nil
		}
		// Otherwise, we will fetch the page from the database
	}
	// Compute the filters
	// We add 6 to skip the system columns
	// Sometimes, a property is not found
	// It can happen when we filter by _page_id, _created_time, etc.
	// Nothing has to be done
	// If we have an int value that is either 0 or 1, we will treat it as a checkbox
	filters := createFilter(constraints, t)

	// Get the rows
	res, err := t.database.Query(context.Background(), notionapi.DatabaseID(t.databaseInfo.ID), &notionapi.DatabaseQueryRequest{
		PageSize:    100,
		StartCursor: notionapi.Cursor(t.nextCursor),
		Filter:      filters,
	})
	if err != nil {
		return nil, false, fmt.Errorf("failed to query database: %w", err)
	}

	rows := make([][]interface{}, 0, len(res.Results))
	for _, page := range res.Results {
		row := make([]interface{}, 0, len(t.columns)+6)
		// Append the user defined columns
		for _, col := range t.columns {
			switch col {
			case "_page_id":
				row = append(row, page.ID.String())
			case "_created_time":
				row = append(row, page.CreatedTime.Format(time.RFC3339))
			case "_last_edited_time":
				row = append(row, page.LastEditedTime.Format(time.RFC3339))
			case "_cover_url":
				if page.Cover != nil {
					row = append(row, page.Cover.GetURL())
				} else {
					row = append(row, nil)
				}
			case "_icon_url":
				if page.Icon != nil {
					row = append(row, page.Icon.GetURL())
				} else {
					row = append(row, nil)
				}
			case "_page_url":
				row = append(row, page.URL)
			default:
				prop, ok := page.Properties[col]
				if !ok {
					row = append(row, nil)
					continue
				}
				row = append(row, unmarshal(prop))
			}
		}
		rows = append(rows, row)
	}
	t.nextCursor = res.NextCursor.String()

	// Store the page in the cache
	value := pageCacheValue{
		Rows:       rows,
		NextCursor: t.nextCursor,
		HasMore:    res.HasMore,
	}

	keyCache := pageCacheKey{
		PageID:     t.page,
		Filter:     constraints.Columns,
		Sort:       constraints.OrderBy,
		Properties: t.databaseInfo.Properties,
	}

	byteCacheKey, err = keyCache.GetKey()
	if err != nil {
		return rows, !res.HasMore, fmt.Errorf("failed to get cache key: %w", err)
	}

	valueBytes, err := json.Marshal(value)
	if err == nil {
		t.cacheDB.Update(func(txn *badger.Txn) error {
			e := badger.NewEntry(byteCacheKey, valueBytes).WithTTL(1 * time.Hour)
			return txn.SetEntry(e)
		})
	}

	return rows, !res.HasMore, nil
}

func createFilter(constraints rpc.QueryConstraint, t *tableCursor) notionapi.AndCompoundFilter {
	filters := notionapi.AndCompoundFilter{}
	for _, constraint := range constraints.Columns {
		columnName := t.columns[constraint.ColumnID]
		property, ok := t.databaseInfo.Properties[columnName]
		if !ok {
			continue
		}
		switch property.GetType() {
		case notionapi.PropertyConfigTypeCheckbox:
			if constraint.Operator == rpc.OperatorEqual {
				if parsed, ok := constraint.Value.(int); ok {
					filters = append(filters, notionapi.PropertyFilter{
						Property: columnName,
						Checkbox: &notionapi.CheckboxFilterCondition{
							Equals: parsed == 1,
						},
					})
				}

			}

		case notionapi.PropertyConfigTypeNumber:
			value := float64(0)
			var ok bool
			if value, ok = constraint.Value.(float64); !ok {
				continue
			}
			var filterNumber *notionapi.NumberFilterCondition
			switch constraint.Operator {
			case rpc.OperatorEqual:
				filterNumber = &notionapi.NumberFilterCondition{
					Equals: &value,
				}
			case rpc.OperatorNotEqual:
				filterNumber = &notionapi.NumberFilterCondition{
					DoesNotEqual: &value,
				}
			case rpc.OperatorGreater:
				filterNumber = &notionapi.NumberFilterCondition{
					GreaterThan: &value,
				}
			case rpc.OperatorGreaterOrEqual:
				filterNumber = &notionapi.NumberFilterCondition{
					GreaterThanOrEqualTo: &value,
				}
			case rpc.OperatorLess:
				filterNumber = &notionapi.NumberFilterCondition{
					LessThan: &value,
				}
			case rpc.OperatorLessOrEqual:
				filterNumber = &notionapi.NumberFilterCondition{
					LessThanOrEqualTo: &value,
				}
			}
			if filterNumber != nil {
				filters = append(filters, notionapi.PropertyFilter{
					Property: columnName,
					Number:   filterNumber,
				})
			}
		case notionapi.PropertyConfigTypeSelect:
			if constraint.Operator == rpc.OperatorEqual {
				if parsed, ok := constraint.Value.(string); ok {
					filters = append(filters, notionapi.PropertyFilter{
						Property: columnName,
						Select: &notionapi.SelectFilterCondition{
							Equals: parsed,
						},
					})
				}
			}
		case notionapi.PropertyConfigStatus:
			if constraint.Operator == rpc.OperatorEqual {
				if parsed, ok := constraint.Value.(string); ok {
					filters = append(filters, notionapi.PropertyFilter{
						Property: columnName,
						Status: &notionapi.StatusFilterCondition{
							Equals: parsed,
						},
					})
				}
			}
		case notionapi.PropertyConfigTypeDate:
			if parsed, ok := constraint.Value.(string); ok {
				splitted := strings.Split(parsed, "/")
				t, err := time.Parse(time.RFC3339, splitted[0])
				notionDate := notionapi.Date(t)
				if err == nil {
					switch constraint.Operator {
					case rpc.OperatorEqual:
						filters = append(filters, notionapi.PropertyFilter{
							Property: columnName,
							Date: &notionapi.DateFilterCondition{
								Equals: &notionDate,
							},
						})
					case rpc.OperatorNotEqual:

					case rpc.OperatorGreater:
						filters = append(filters, notionapi.PropertyFilter{
							Property: columnName,
							Date: &notionapi.DateFilterCondition{
								After: &notionDate,
							},
						})

					case rpc.OperatorLess:
						filters = append(filters, notionapi.PropertyFilter{
							Property: columnName,
							Date: &notionapi.DateFilterCondition{
								Before: &notionDate,
							},
						})

					case rpc.OperatorGreaterOrEqual:
						filters = append(filters, notionapi.PropertyFilter{
							Property: columnName,
							Date: &notionapi.DateFilterCondition{
								OnOrAfter: &notionDate,
							},
						})

					case rpc.OperatorLessOrEqual:
						filters = append(filters, notionapi.PropertyFilter{
							Property: columnName,
							Date: &notionapi.DateFilterCondition{
								OnOrBefore: &notionDate,
							},
						})

					}
				}
			}

		case notionapi.PropertyConfigTypeRichText, notionapi.PropertyConfigTypeTitle, notionapi.PropertyConfigTypeEmail, notionapi.PropertyConfigTypePhoneNumber, notionapi.PropertyConfigTypeURL:
			if parsed, ok := constraint.Value.(string); ok {
				switch constraint.Operator {
				case rpc.OperatorEqual:
					filters = append(filters, notionapi.PropertyFilter{
						Property: columnName,
						RichText: &notionapi.TextFilterCondition{
							Equals: parsed,
						},
					})
				case rpc.OperatorNotEqual:
					filters = append(filters, notionapi.PropertyFilter{
						Property: columnName,
						RichText: &notionapi.TextFilterCondition{
							DoesNotEqual: parsed,
						},
					})
				case rpc.OperatorGlob:
					if strings.HasPrefix(parsed, "*") && strings.HasSuffix(parsed, "*") {
						filters = append(filters, notionapi.PropertyFilter{
							Property: columnName,
							RichText: &notionapi.TextFilterCondition{
								Contains: strings.ReplaceAll(parsed, "*", ""),
							},
						})
					} else if strings.HasSuffix(parsed, "*") {
						filters = append(filters, notionapi.PropertyFilter{
							Property: columnName,
							RichText: &notionapi.TextFilterCondition{
								StartsWith: strings.ReplaceAll(parsed, "*", ""),
							},
						})
					} else if strings.HasPrefix(parsed, "*") {
						filters = append(filters, notionapi.PropertyFilter{
							Property: columnName,
							RichText: &notionapi.TextFilterCondition{
								EndsWith: strings.ReplaceAll(parsed, "*", ""),
							},
						})
					} else {
						filters = append(filters, notionapi.PropertyFilter{
							Property: columnName,
							RichText: &notionapi.TextFilterCondition{
								Contains: strings.ReplaceAll(parsed, "*", ""),
							},
						})
					}
				}

			}
		case notionapi.PropertyConfigTypeFormula:
			switch constraint.Value.(type) {
			case string:
				switch constraint.Operator {
				case rpc.OperatorEqual:
					filters = append(filters, notionapi.PropertyFilter{
						Property: columnName,
						Formula: &notionapi.FormulaFilterCondition{
							Text: &notionapi.TextFilterCondition{
								Equals: constraint.Value.(string),
							},
						},
					})
				case rpc.OperatorNotEqual:
					filters = append(filters, notionapi.PropertyFilter{
						Property: columnName,
						Formula: &notionapi.FormulaFilterCondition{
							Text: &notionapi.TextFilterCondition{
								DoesNotEqual: constraint.Value.(string),
							},
						},
					})

				case rpc.OperatorLike:
					filters = append(filters, notionapi.PropertyFilter{
						Property: columnName,
						Formula: &notionapi.FormulaFilterCondition{
							Text: &notionapi.TextFilterCondition{
								Contains: constraint.Value.(string),
							},
						},
					})
				}
			case float64, int64:
				var value float64
				if _, ok := constraint.Value.(int64); ok {
					value = float64(constraint.Value.(int64))

					if value == 0 || value == 1 {
						filters = append(filters, notionapi.PropertyFilter{
							Property: columnName,
							Formula: &notionapi.FormulaFilterCondition{
								Checkbox: &notionapi.CheckboxFilterCondition{
									Equals: value == 1,
								},
							},
						})
						break
					}
				} else {
					value = constraint.Value.(float64)
				}

				var filterNumber *notionapi.NumberFilterCondition
				switch constraint.Operator {
				case rpc.OperatorEqual:
					filterNumber = &notionapi.NumberFilterCondition{
						Equals: &value,
					}
				case rpc.OperatorNotEqual:
					filterNumber = &notionapi.NumberFilterCondition{
						DoesNotEqual: &value,
					}
				case rpc.OperatorGreater:
					filterNumber = &notionapi.NumberFilterCondition{
						GreaterThan: &value,
					}
				case rpc.OperatorGreaterOrEqual:
					filterNumber = &notionapi.NumberFilterCondition{
						GreaterThanOrEqualTo: &value,
					}
				case rpc.OperatorLess:
					filterNumber = &notionapi.NumberFilterCondition{
						LessThan: &value,
					}
				case rpc.OperatorLessOrEqual:
					filterNumber = &notionapi.NumberFilterCondition{
						LessThanOrEqualTo: &value,
					}

				}
				if filterNumber != nil {
					filters = append(filters, notionapi.PropertyFilter{
						Property: columnName,
						Number:   filterNumber,
					})
				}
			}
		}
	}
	return filters
}

func (t *table) CreateReader() rpc.ReaderInterface {
	return &tableCursor{"", t.dbClient, t.database, t.columns, 0, t.cacheDB}
}

func (t *table) Close() error {
	return t.cacheDB.Close()
}
