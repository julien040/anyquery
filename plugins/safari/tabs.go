//go:build darwin

package main

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"os/exec"

	"github.com/julien040/anyquery/rpc"
)

//go:embed scripts/listTabs.js
var listTabsScript string

//go:embed scripts/newTab.applescript
var newTabScript string

//go:embed scripts/activateTab.applescript
var activateTabScript string

//go:embed scripts/setURL.js
var setURLScript string

// A constructor to create a new table instance
// This function is called everytime a new connection is made to the plugin
//
// It should return a new table instance, the database schema and if there is an error
func tabsCreator(args rpc.TableCreatorArgs) (rpc.Table, *rpc.DatabaseSchema, error) {
	return &tabsTable{}, &rpc.DatabaseSchema{
		PrimaryKey:    6,
		HandlesInsert: true,
		HandlesUpdate: true,
		HandlesDelete: false, // Due to index issues, tabs deletion is delayed
		Columns: []rpc.DatabaseSchemaColumn{
			{
				Name: "tab_index",
				Type: rpc.ColumnTypeInt,
			},
			{
				Name: "title",
				Type: rpc.ColumnTypeString,
			},
			{
				Name: "url",
				Type: rpc.ColumnTypeString,
			},
			{
				Name: "window_name",
				Type: rpc.ColumnTypeString,
			},
			{
				Name: "window_index",
				Type: rpc.ColumnTypeInt,
			},
			{
				Name: "visible",
				Type: rpc.ColumnTypeInt,
			},
			{
				Name: "uid",
				Type: rpc.ColumnTypeInt,
			},
		},
	}, nil
}

type tabsTable struct {
}

type tabsCursor struct {
}

type item struct {
	Index       int    `json:"index"`
	Title       string `json:"title"`
	URL         string `json:"url"`
	WindowName  string `json:"windowName"`
	WindowIndex int    `json:"windowIndex"`
	Visible     bool   `json:"visible"`
}

// Return a slice of rows that will be returned to Anyquery and filtered.
// The second return value is true if the cursor has no more rows to return
//
// The constraints are used for optimization purposes to "pre-filter" the rows
// If the rows returned don't match the constraints, it's not an issue. Anyquery will filter them out
func (t *tabsCursor) Query(constraints rpc.QueryConstraint) ([][]interface{}, bool, error) {
	// Run the script to get the tabs
	cmd := exec.Command("osascript", "-l", "JavaScript", "-e", listTabsScript)
	out, err := cmd.StderrPipe()
	if err != nil {
		return nil, true, fmt.Errorf("can't get stderr pipe: %w", err)
	}

	jsonDecoder := json.NewDecoder(out)
	if err := cmd.Start(); err != nil {
		return nil, true, fmt.Errorf("can't start command: %w", err)
	}
	item := item{}
	rows := make([][]interface{}, 0)
	for {
		if err := jsonDecoder.Decode(&item); err != nil {
			break
		}
		rows = append(rows, []interface{}{
			item.Index,
			item.Title,
			item.URL,
			item.WindowName,
			item.WindowIndex,
			item.Visible,
			// We set a hidden primary key that will be used by the delete and update functions
			// to retrieve at the same time the index of the tab and the window index
			item.Index + item.WindowIndex*10000,
		})
	}

	if err := cmd.Wait(); err != nil {
		return nil, true, fmt.Errorf("can't wait for command: %w", err)
	}

	return rows, true, nil
}

// Create a new cursor that will be used to read rows
func (t *tabsTable) CreateReader() rpc.ReaderInterface {
	return &tabsCursor{}
}

// A slice of rows to insert
func (t *tabsTable) Insert(rows [][]interface{}) error {
	for _, row := range rows {
		windowIndex := 1
		if rawInt, ok := row[4].(int64); ok {
			windowIndex = int(rawInt)
		}

		url := "favorites://"
		if rawURL, ok := row[2].(string); ok {
			url = rawURL
		}

		cmd := exec.Command("osascript", "-e", fmt.Sprintf(newTabScript, url, windowIndex))
		output, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("can't run osascript: %W (message: %s)\n Script: %s", err, output, fmt.Sprintf(newTabScript, windowIndex, url))
		}

	}

	return nil
}

// A slice of rows to update
// The first element of each row is the primary key
// while the rest are the values to update
// The primary key is therefore present twice
func (t *tabsTable) Update(rows [][]interface{}) error {
	for _, row := range rows {
		index := 1
		window := 1
		if rawInt, ok := row[0].(int64); ok { // The pk is at index 0 on update
			index = int(rawInt) % 10000
			window = int(rawInt) / 10000
		}

		url := ""
		if rawURL, ok := row[3].(string); ok {
			url = rawURL
		}

		if url != "" {
			cmd := exec.Command("osascript", "-l", "JavaScript", "-e", fmt.Sprintf(setURLScript, window, index, url))
			output, err := cmd.CombinedOutput()
			if err != nil {
				return fmt.Errorf("can't run osascript: %W (message: %s)\n Script: %s", err, output, fmt.Sprintf(setURLScript, window, index, url))
			}
		}

		// Activate the tab if needed
		if rawVisible, ok := row[6].(int64); ok && rawVisible == 1 {
			cmd := exec.Command("osascript", "-e", fmt.Sprintf(activateTabScript, window, index))
			output, err := cmd.CombinedOutput()
			if err != nil {
				return fmt.Errorf("can't run osascript: %W (message: %s)\n Script: %s", err, output, fmt.Sprintf(activateTabScript, window, index))
			}
		}
	}

	return nil

}

// A slice of primary keys to delete
func (t *tabsTable) Delete(primaryKeys []interface{}) error {
	return fmt.Errorf("deletion of tabs is not yet supported")
}

// A destructor to clean up resources
func (t *tabsTable) Close() error {
	return nil
}
