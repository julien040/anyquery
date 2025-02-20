package main

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	"github.com/julien040/anyquery/rpc"
)

// A constructor to create a new table instance
// This function is called everytime a new connection is made to the plugin
//
// It should return a new table instance, the database schema and if there is an error
func downloadsCreator(args rpc.TableCreatorArgs) (rpc.Table, *rpc.DatabaseSchema, error) {
	if databaseHistoryPath == "" {
		return nil, nil, fmt.Errorf("failed to find the history database path")
	}

	db, err := sql.Open("sqlite", fmt.Sprintf("file:%s?mode=ro", databaseHistoryPath))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to find the history database: %w", err)
	}

	return &downloadsTable{
			db: db,
		}, &rpc.DatabaseSchema{
			Columns: []rpc.DatabaseSchemaColumn{
				{
					Name:        "id",
					Type:        rpc.ColumnTypeInt,
					Description: "The ID of the download",
				},
				{
					Name:        "path",
					Type:        rpc.ColumnTypeString,
					Description: "The path where the download is stored",
				},
				{
					Name:        "started_at",
					Type:        rpc.ColumnTypeDateTime,
					Description: "The datetime (RFC3339) when the download started",
				},
				{
					Name:        "ended_at",
					Type:        rpc.ColumnTypeDateTime,
					Description: "The datetime (RFC3339) when the download ended",
				},
				{
					Name:        "url",
					Type:        rpc.ColumnTypeString,
					Description: "The URL of the download",
				},
				{
					Name:        "size",
					Type:        rpc.ColumnTypeInt,
					Description: "The size of the download in bytes",
				},
				{
					Name:        "mime_type",
					Type:        rpc.ColumnTypeString,
					Description: "The MIME type of the download",
				},
			},
		}, nil
}

// The table struct
// There is one per connection to the plugin and is created by the creator function
// In there, you can store any state you need to read the rows (e.g. a database connection, an API token, etc.)
type downloadsTable struct {
	db *sql.DB
}

// The cursor struct
// There is one per query and is created by the createReader function
// In there, you can store any state you need to read the rows (e.g. a database connection from downloadsTable, an offset, a cursor, etc.)
type downloadsCursor struct {
	db *sql.DB
}

// Create a new cursor that will be used to read rows
func (t *downloadsTable) CreateReader() rpc.ReaderInterface {
	return &downloadsCursor{
		db: t.db,
	}
}

const queryDownloads = `
SELECT
	id,
	CURRENT_PATH,
	start_time,
	end_time,
	tab_url,
	total_bytes AS size,
	mime_type
FROM
	downloads
ORDER BY
	start_time DESC;
`

// Return a slice of rows that will be returned to Anyquery and filtered.
// The second return value is true if the cursor has no more rows to return
//
// The constraints are used for optimization purposes to "pre-filter" the rows
// If the rows returned don't match the constraints, it's not an issue. Anyquery will filter them out
func (t *downloadsCursor) Query(constraints rpc.QueryConstraint) ([][]interface{}, bool, error) {

	rows, err := t.db.Query(queryDownloads)
	if err != nil {
		return nil, true, fmt.Errorf("failed to query downloads: %w", err)
	}

	var result [][]interface{}
	for rows.Next() {
		var id, size int64
		var path, url, mimeType string
		var startedAt int64
		var endedAt sql.NullInt64

		err := rows.Scan(&id, &path, &startedAt, &endedAt, &url, &size, &mimeType)
		if err != nil {
			return nil, true, fmt.Errorf("failed to scan downloads: %w", err)
		}

		// Convert the WebKit timestamp to a RFC3339 datetime
		startedAtStr := time.Unix(startedAt/1000000-11644473600, 0).Format(time.RFC3339)
		endAtStr := interface{}(nil)
		if endedAt.Valid {
			endAtStr = time.Unix(endedAt.Int64/1000000-11644473600, 0).Format(time.RFC3339)
		}

		log.Printf("id: %d, path: %s, startedAt: %s, endedAt: %s, url: %s, size: %d, mimeType: %s", id, path, startedAtStr, endAtStr, url, size, mimeType)

		result = append(result, []interface{}{
			id,
			path,
			startedAtStr,
			endAtStr,
			url,
			size,
			mimeType,
		})
	}

	return result, true, nil
}

// A destructor to clean up resources
func (t *downloadsTable) Close() error {
	return t.db.Close()
}
