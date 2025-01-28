package main

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/gob"
	"fmt"
	"log"
	"os"
	"path"
	"strings"
	"time"

	"github.com/adrg/xdg"
	"github.com/dgraph-io/badger/v4"
	"github.com/hashicorp/go-retryablehttp"
	"github.com/julien040/anyquery/rpc"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/option"
	"google.golang.org/api/sheets/v4"
)

var retryableClient = retryablehttp.NewClient()

// A constructor to create a new table instance
// This function is called everytime a new connection is made to the plugin
//
// It should return a new table instance, the database schema and if there is an error
func tableCreator(args rpc.TableCreatorArgs) (rpc.Table, *rpc.DatabaseSchema, error) {
	var spreadsheetID, token, clientID, clientSecret, sheetName string
	var sheetID int64
	var caching bool
	if rawInter, ok := args.UserConfig["spreadsheet_id"]; ok {
		if spreadsheetID, ok = rawInter.(string); !ok {
			return nil, nil, fmt.Errorf("spreadsheet_id should be a string")
		}
		if spreadsheetID == "" {
			return nil, nil, fmt.Errorf("spreadsheet_id should not be empty")
		}
	} else {
		return nil, nil, fmt.Errorf("spreadsheet_id is required")
	}

	if rawInter, ok := args.UserConfig["token"]; ok {
		if token, ok = rawInter.(string); !ok {
			return nil, nil, fmt.Errorf("token should be a string")
		}
		if token == "" {
			return nil, nil, fmt.Errorf("token should not be empty")
		}
	} else {
		return nil, nil, fmt.Errorf("token is required")
	}

	if rawInter, ok := args.UserConfig["client_id"]; ok {
		if clientID, ok = rawInter.(string); !ok {
			return nil, nil, fmt.Errorf("client_id should be a string")
		}
		if clientID == "" {
			return nil, nil, fmt.Errorf("client_id should not be empty")
		}
	} else {
		return nil, nil, fmt.Errorf("client_id is required")
	}

	if rawInter, ok := args.UserConfig["client_secret"]; ok {
		if clientSecret, ok = rawInter.(string); !ok {
			return nil, nil, fmt.Errorf("client_secret should be a string")
		}
		if clientSecret == "" {
			return nil, nil, fmt.Errorf("client_secret should not be empty")
		}
	} else {
		return nil, nil, fmt.Errorf("client_secret is required")
	}

	// Optional
	if rawInter, ok := args.UserConfig["sheet_name"]; ok {
		if sheetName, ok = rawInter.(string); !ok {
			return nil, nil, fmt.Errorf("sheet_id should be a string")
		}
	}

	// Optional
	if rawInter, ok := args.UserConfig["caching"]; ok {
		if caching, ok = rawInter.(bool); !ok {
			return nil, nil, fmt.Errorf("caching should be a boolean")
		}
	}

	config := oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		Endpoint:     google.Endpoint,
		Scopes:       []string{sheets.SpreadsheetsScope},
	}

	oauthClient := config.Client(context.Background(), &oauth2.Token{
		RefreshToken: token,
	})

	retryableClient = retryablehttp.NewClient()
	retryableClient.HTTPClient = oauthClient

	// Fetch the spreadsheet
	srv, err := sheets.NewService(context.Background(), option.WithHTTPClient(retryableClient.StandardClient()))
	if err != nil {
		return nil, nil, fmt.Errorf("unable to retrieve Sheets client: %v", err)
	}

	req := srv.Spreadsheets.Get(spreadsheetID)
	/* req.Ranges("1:2") // Only fetch the first two rows to get the schema and the column types
	req.IncludeGridData(true) */
	spreadsheet, err := req.Do()
	if err != nil {
		return nil, nil, fmt.Errorf("unable to retrieve spreadsheet: %v", err)
	}

	if len(spreadsheet.Sheets) == 0 {
		return nil, nil, fmt.Errorf("no sheets found in the spreadsheet")
	}

	sheetID = -1

	if sheetName == "" {
		sheetName = spreadsheet.Sheets[0].Properties.Title
		sheetID = spreadsheet.Sheets[0].Properties.SheetId
	} else {
		for _, sheet := range spreadsheet.Sheets {
			if sheet.Properties.Title == sheetName {
				sheetID = sheet.Properties.SheetId
				break
			}
		}
		if sheetID == -1 {
			return nil, nil, fmt.Errorf("sheet %s not found in the spreadsheet", sheetName)
		}
	}

	// Get the first two rows to get the schema
	req = srv.Spreadsheets.Get(spreadsheetID)
	req.Ranges(fmt.Sprintf("%s!1:2", sheetName))
	req.IncludeGridData(true)
	spreadsheet, err = req.Do()
	if err != nil {
		return nil, nil, fmt.Errorf("unable to retrieve spreadsheet: %v", err)
	}

	if len(spreadsheet.Sheets) == 0 {
		return nil, nil, fmt.Errorf("no sheets found in the spreadsheet")
	}

	// Compute the schema
	schema := []rpc.DatabaseSchemaColumn{
		{
			Name:        "rowIndex",
			Type:        rpc.ColumnTypeInt,
			Description: "The row index in the spreadsheet",
		},
	}

	// On the second call, we only fetch the right sheet because we are passing a range
	if len(spreadsheet.Sheets[0].Data) == 0 {
		return nil, nil, fmt.Errorf("no data found in the spreadsheet")
	}

	mapColIndex := make(map[int]int)
	mapColIndexReverse := make(map[int]int)
	sqliteIndex := 1 // Start at 1 because the first column is the primary key
	for columnIndex, cell := range spreadsheet.Sheets[0].Data[0].RowData[0].Values {
		if cell.FormattedValue == "" {
			continue
		}
		colType := rpc.ColumnTypeFloat // Default type that will be converted to string or int if needed by SQLite
		if len(spreadsheet.Sheets[0].Data[0].RowData) > 1 && columnIndex < len(spreadsheet.Sheets[0].Data[0].RowData[1].Values) {
			val := spreadsheet.Sheets[0].Data[0].RowData[1].Values[columnIndex]
			if val.EffectiveValue != nil {
				switch {
				case val.EffectiveValue.BoolValue != nil:
					colType = rpc.ColumnTypeBool
				case val.EffectiveValue.NumberValue != nil:
					// Check if it's a date
					if val.EffectiveFormat.NumberFormat != nil {
						if val.EffectiveFormat.NumberFormat.Type == "DATE" || val.EffectiveFormat.NumberFormat.Type == "TIME" ||
							val.EffectiveFormat.NumberFormat.Type == "DATE_TIME" {
							colType = rpc.ColumnTypeString
						}
					} else {
						colType = rpc.ColumnTypeFloat
					}
				case val.EffectiveValue.StringValue != nil:
					colType = rpc.ColumnTypeString
				case val.EffectiveValue.ErrorValue != nil:
					colType = rpc.ColumnTypeString
				default: // In anyquery, a float can be converted to anything
					colType = rpc.ColumnTypeFloat
				}
			}
		}
		mapColIndex[columnIndex] = sqliteIndex
		mapColIndexReverse[sqliteIndex] = columnIndex
		schema = append(schema, rpc.DatabaseSchemaColumn{
			Name:        cell.FormattedValue,
			Type:        colType,
			Description: fmt.Sprintf("The %dth column in the spreadsheet", columnIndex+1),
		})
		sqliteIndex++

	}

	// If the cache is enabled, open the badger database
	var db *badger.DB

	if caching {
		hashedToken := md5.Sum([]byte(token))
		cacheFolder := path.Join(xdg.CacheHome, "anyquery", "plugins", "notion", fmt.Sprintf("%x", hashedToken[:]), spreadsheetID, fmt.Sprintf("%d", sheetID))
		err = os.MkdirAll(cacheFolder, 0755)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to create cache folder: %w", err)
		}

		// Open the badger database encrypted with the toke
		options := badger.DefaultOptions(cacheFolder).WithEncryptionKey(hashedToken[:]).
			WithNumVersionsToKeep(1).WithCompactL0OnClose(true).WithValueLogFileSize(2 << 24).
			WithIndexCacheSize(2 << 23)
		badgerDB, err := badger.Open(options)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to open badger database: %w", err)
		}
		db = badgerDB
	}

	tableDesc := strings.Builder{}
	if spreadsheet.Properties != nil {
		tableDesc.WriteString(fmt.Sprintf("Spreadsheet %s", spreadsheet.Properties.Title))
		tableDesc.WriteString(fmt.Sprintf(" using locale %s", spreadsheet.Properties.Locale))
	}

	if spreadsheet.Sheets[0].Properties != nil {
		tableDesc.WriteString(fmt.Sprintf(" and sheet %s", spreadsheet.Sheets[0].Properties.Title))
	}

	return &tableTable{
			spreadsheetID:      spreadsheetID,
			sheetID:            sheetID,
			sheetName:          sheetName,
			srv:                srv,
			cacheDB:            db,
			mapColIndex:        mapColIndex,
			mapColIndexReverse: mapColIndexReverse,
		}, &rpc.DatabaseSchema{
			HandlesInsert: true,
			HandlesUpdate: true,
			HandlesDelete: true,
			HandleOffset:  false,
			BufferInsert:  200,
			BufferUpdate:  200,
			BufferDelete:  200,
			Columns:       schema,
			Description:   tableDesc.String(),
		}, nil
}

type tableTable struct {
	spreadsheetID      string
	sheetID            int64
	sheetName          string
	srv                *sheets.Service
	cacheDB            *badger.DB
	mapColIndex        map[int]int
	mapColIndexReverse map[int]int
}

type tableCursor struct {
	spreadsheetID      string
	sheetID            int64
	sheetName          string
	srv                *sheets.Service
	cacheDB            *badger.DB
	mapColIndex        map[int]int
	mapColIndexReverse map[int]int
	offset             int
	pageSize           int
}

// Return a slice of rows that will be returned to Anyquery and filtered.
// The second return value is true if the cursor has no more rows to return
//
// The constraints are used for optimization purposes to "pre-filter" the rows
// If the rows returned don't match the constraints, it's not an issue. Anyquery will filter them out
func (t *tableCursor) Query(constraints rpc.QueryConstraint) ([][]interface{}, bool, error) {
	// Request the rows from the spreadsheet or from the cache
	rangeStr := fmt.Sprintf("%s!%d:%d", t.sheetName, t.offset, t.offset+t.pageSize)

	rows := make([][]interface{}, 0)
	if t.cacheDB != nil {
		// Check if the data is in the cache
		err := t.cacheDB.View(func(txn *badger.Txn) error {
			item, err := txn.Get([]byte(rangeStr))
			if err != nil {
				return err
			}
			// Decode the rows with gob
			return item.Value(func(val []byte) error {
				decoder := gob.NewDecoder(bytes.NewReader(val))
				return decoder.Decode(&rows)
			})
		})
		if err == nil {
			// Update the offset
			t.offset += t.pageSize
			// Return the rows
			return rows, len(rows) < t.pageSize, nil
		}
	}

	req := t.srv.Spreadsheets.Values.Get(t.spreadsheetID, rangeStr)
	req.DateTimeRenderOption("FORMATTED_STRING")
	req.MajorDimension("ROWS")
	req.ValueRenderOption("UNFORMATTED_VALUE")

	resp, err := req.Do()
	if err != nil {
		return nil, true, fmt.Errorf("unable to retrieve data from the spreadsheet: %v", err)
	}

	if len(resp.Values) == 0 {
		return nil, true, nil
	}

	rows = make([][]interface{}, 0, len(resp.Values))
	for i, row := range resp.Values {
		rowIndex := t.offset + i
		newRow := make([]interface{}, len(t.mapColIndex)+1)
		newRow[0] = rowIndex
		for colIndex, col := range row {
			if sqliteIndex, ok := t.mapColIndex[colIndex]; ok {
				newRow[sqliteIndex] = col
			}
		}
		rows = append(rows, newRow)
	}

	// Save the rows in the cache
	if t.cacheDB != nil {
		err := t.cacheDB.Update(func(txn *badger.Txn) error {
			var buf bytes.Buffer
			encoder := gob.NewEncoder(&buf)
			err := encoder.Encode(rows)
			if err != nil {
				return err
			}
			e := badger.NewEntry([]byte(rangeStr), buf.Bytes()).WithTTL(time.Hour)
			return txn.SetEntry(e)
		})
		if err != nil {
			log.Printf("Failed to save data in cache: %v", err)
		}
	}

	// Update the offset
	t.offset += t.pageSize

	// Return the rows
	return rows, len(resp.Values) < t.pageSize, nil
}

// Create a new cursor that will be used to read rows
func (t *tableTable) CreateReader() rpc.ReaderInterface {
	return &tableCursor{
		spreadsheetID:      t.spreadsheetID,
		sheetID:            t.sheetID,
		sheetName:          t.sheetName,
		srv:                t.srv,
		cacheDB:            t.cacheDB,
		mapColIndex:        t.mapColIndex,
		mapColIndexReverse: t.mapColIndexReverse,
		offset:             2,
		pageSize:           2000,
	}
}

// A slice of rows to insert
func (t *tableTable) Insert(rows [][]interface{}) error {
	// We will append the rows to the end of the sheet
	request := make([]*sheets.Request, 0, len(rows))
	for _, row := range rows {
		values := make([]*sheets.CellData, len(row)-1)
		for i, cell := range row[1:] {
			extendedValue := &sheets.ExtendedValue{}
			switch typed := cell.(type) {
			case string:
				extendedValue.StringValue = &typed
			case int64:
				asFloat := float64(typed)
				extendedValue.NumberValue = &asFloat
			case float64:
				extendedValue.NumberValue = &typed
			}
			colIndex, ok := t.mapColIndexReverse[i+1]
			if !ok {
				return fmt.Errorf("unable to find column index for column %d", i+1)
			}
			values[colIndex] = &sheets.CellData{
				UserEnteredValue: extendedValue,
			}
		}
		request = append(request, &sheets.Request{
			AppendCells: &sheets.AppendCellsRequest{
				SheetId: t.sheetID,
				Rows: []*sheets.RowData{
					{
						Values: values,
					},
				},
				Fields: "*",
			},
		})

	}

	call := t.srv.Spreadsheets.BatchUpdate(t.spreadsheetID, &sheets.BatchUpdateSpreadsheetRequest{
		Requests: request,
	})

	_, err := call.Do()
	if err != nil {
		return fmt.Errorf("unable to insert rows: %v", err)
	}

	return t.clearCache()
}

// A slice of rows to update
// The first element of each row is the primary key
// while the rest are the values to update
// The primary key is therefore present twice
func (t *tableTable) Update(rows [][]interface{}) error {
	request := make([]*sheets.Request, 0, len(rows))
	for _, row := range rows {
		values := make([]*sheets.CellData, len(row)-1)
		for i, cell := range row[2:] { // The first element is the former primary key, the second is the new primary key
			extendedValue := &sheets.ExtendedValue{}
			switch typed := cell.(type) {
			case string:
				extendedValue.StringValue = &typed
			case int64:
				asFloat := float64(typed)
				extendedValue.NumberValue = &asFloat
			case float64:
				extendedValue.NumberValue = &typed
			}
			colIndex, ok := t.mapColIndexReverse[i+1]
			if !ok {
				return fmt.Errorf("unable to find column index for column %d", i+1)
			}
			values[colIndex] = &sheets.CellData{
				UserEnteredValue: extendedValue,
			}
		}
		request = append(request, &sheets.Request{
			UpdateCells: &sheets.UpdateCellsRequest{
				Rows: []*sheets.RowData{
					{
						Values: values,
					},
				},
				Start: &sheets.GridCoordinate{
					SheetId:     t.sheetID,
					RowIndex:    (row[0].(int64)) - 1,           // The row index starts to write at the row before the one we want to update
					ColumnIndex: int64(t.mapColIndexReverse[1]), // Index of the first column
				},
				Fields: "*",
			},
		})

	}

	call := t.srv.Spreadsheets.BatchUpdate(t.spreadsheetID, &sheets.BatchUpdateSpreadsheetRequest{
		Requests: request,
	})
	_, err := call.Do()
	if err != nil {
		return fmt.Errorf("unable to update rows: %v", err)
	}

	return t.clearCache()

}

// A slice of primary keys to delete
func (t *tableTable) Delete(primaryKeys []interface{}) error {
	request := make([]*sheets.Request, 0, len(primaryKeys))
	for i, primaryKey := range primaryKeys {
		request = append(request, &sheets.Request{
			DeleteDimension: &sheets.DeleteDimensionRequest{
				Range: &sheets.DimensionRange{
					SheetId:    t.sheetID,
					Dimension:  "ROWS",
					StartIndex: (primaryKey.(int64)) - 1 - int64(i), // The row index starts to delete at the row before the one we want to delete
					EndIndex:   (primaryKey.(int64)) - int64(i),     // Because delete are successive, we need to subtract the number of rows already deleted
				},
			},
		})
	}

	call := t.srv.Spreadsheets.BatchUpdate(t.spreadsheetID, &sheets.BatchUpdateSpreadsheetRequest{
		Requests: request,
	})
	_, err := call.Do()
	if err != nil {
		return fmt.Errorf("unable to delete rows: %v", err)
	}
	return t.clearCache()
}

// A destructor to clean up resources
func (t *tableTable) Close() error {
	return nil
}

func (t *tableTable) clearCache() error {
	if t.cacheDB != nil {
		err := t.cacheDB.DropAll()
		if err != nil {
			return fmt.Errorf("unable to clear cache: %v", err)
		}
	}
	return nil
}
