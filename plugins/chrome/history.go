package main

import (
	"database/sql"
	"fmt"
	"os"
	"path"
	"runtime"

	"github.com/julien040/anyquery/rpc"

	_ "modernc.org/sqlite"
)

var databaseHistoryPath = ""

func init() {
	// Update the database history path following the OS
	switch runtime.GOOS {
	case "windows":
		// Get the appdata directory
		appData := os.Getenv("APPDATA")
		if appData == "" {
			return
		}

		databaseHistoryPath = path.Join(appData, "local", "Google", "Chrome", "User Data", "Default", "History")
	case "darwin":
		// Get the home directory
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return
		}

		databaseHistoryPath = path.Join(homeDir, "Library", "Application Support", "Google", "Chrome", "Default", "History")
	case "linux":
		// Get the home directory
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return
		}

		databaseHistoryPath = path.Join(homeDir, ".config", "google-chrome", "Default", "History")
	}
}

// A constructor to create a new table instance
// This function is called everytime a new connection is made to the plugin
//
// It should return a new table instance, the database schema and if there is an error
func historyCreator(args rpc.TableCreatorArgs) (rpc.Table, *rpc.DatabaseSchema, error) {
	if databaseHistoryPath == "" {
		return nil, nil, fmt.Errorf("failed to find the history database path")
	}

	db, err := sql.Open("sqlite", fmt.Sprintf("file:%s?mode=ro", databaseHistoryPath))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to find the history database: %w", err)
	}

	return &historyTable{
			db: db,
		}, &rpc.DatabaseSchema{
			Columns: []rpc.DatabaseSchemaColumn{
				{
					Name:        "url",
					Type:        rpc.ColumnTypeString,
					Description: "The URL of the visited page",
				},
				{
					Name:        "visited_at",
					Type:        rpc.ColumnTypeDateTime,
					Description: "The date and time the page was visited",
				},
				{
					Name:        "visited_for_milliseconds",
					Type:        rpc.ColumnTypeInt,
					Description: "The time spent on the page in milliseconds",
				},
				{
					Name:        "title",
					Type:        rpc.ColumnTypeString,
					Description: "The title of the visited page",
				},
			},
		}, nil
}

// The table struct
// There is one per connection to the plugin and is created by the creator function
// In there, you can store any state you need to read the rows (e.g. a database connection, an API token, etc.)
type historyTable struct {
	db *sql.DB
}

// The cursor struct
// There is one per query and is created by the createReader function
// In there, you can store any state you need to read the rows (e.g. a database connection from historyTable, an offset, a cursor, etc.)
type historyCursor struct {
	db *sql.DB
}

// Create a new cursor that will be used to read rows
func (t *historyTable) CreateReader() rpc.ReaderInterface {
	return &historyCursor{
		db: t.db,
	}
}

const query = `
SELECT
	u.url,
	datetime((visit_time / 1000000) - 11644473600, 'unixepoch') AS visited_at,
	visit_duration / 1000 AS visited_for_milliseconds,
	u.title
FROM
	urls u,
	visits v
WHERE
	u.id = v.url
ORDER BY
	visited_at DESC;
`

// Return a slice of rows that will be returned to Anyquery and filtered.
// The second return value is true if the cursor has no more rows to return
//
// The constraints are used for optimization purposes to "pre-filter" the rows
// If the rows returned don't match the constraints, it's not an issue. Anyquery will filter them out
func (t *historyCursor) Query(constraints rpc.QueryConstraint) ([][]interface{}, bool, error) {
	if t.db == nil {
		return nil, true, fmt.Errorf("database connection is nil")
	}

	rows, err := t.db.Query(query)
	if err != nil {
		return nil, true, fmt.Errorf("failed to query the history database: %w", err)
	}

	var result [][]interface{}
	for rows.Next() {
		var url, visitedAt, title string
		var visitedForMilliseconds int64
		err := rows.Scan(&url, &visitedAt, &visitedForMilliseconds, &title)
		if err != nil {
			return nil, true, fmt.Errorf("failed to scan the history database: %w", err)
		}

		result = append(result, []interface{}{url, visitedAt, visitedForMilliseconds, title})
	}

	if err := rows.Err(); err != nil {
		return nil, true, fmt.Errorf("failed to query the history database: %w", err)
	}

	return result, true, nil
}

// A slice of rows to insert
// Uncomment the code to add support for inserting rows
/*
func (t *historyTable) Insert(rows [][]interface{}) error {
	// Example: insert the rows in a database
	// for _, row := range rows {
	// 	err := db.Insert(row[0], row[1], row[2])
	// 	if err != nil {
	// 		return err
	// 	}
	return nil
}
*/

// A slice of rows to update
// The first element of each row is the primary key
// while the rest are the values to update
// The primary key is therefore present twice
// Uncomment the code to add support for updating rows
/*
func (t *historyTable) Update(rows [][]interface{}) error {
	return nil
}*/

// A slice of primary keys to delete
// Uncomment the code to add support for deleting rows
/*
func (t *historyTable) Delete(primaryKeys []interface{}) error {
	return nil
}
*/

// A destructor to clean up resources
func (t *historyTable) Close() error {
	return t.db.Close()
}
