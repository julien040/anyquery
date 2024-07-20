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

//go:embed scripts/setURL.js
var setURLScript string

//go:embed scripts/deleteTab.js
var deleteTabScript string

// A constructor to create a new table instance
// This function is called everytime a new connection is made to the plugin
//
// It should return a new table instance, the database schema and if there is an error
func tabsCreator(args rpc.TableCreatorArgs) (rpc.Table, *rpc.DatabaseSchema, error) {
	return &tabsTable{}, &rpc.DatabaseSchema{
		PrimaryKey:    0,
		HandlesInsert: true,
		HandlesUpdate: true,
		HandlesDelete: true,
		Columns: []rpc.DatabaseSchemaColumn{
			{
				Name: "id",
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
				Name: "window_id",
				Type: rpc.ColumnTypeInt,
			},
			{
				Name: "active",
				Type: rpc.ColumnTypeInt,
			},
			{
				Name: "loading",
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
	ID         string `json:"id"`
	Title      string `json:"title"`
	URL        string `json:"url"`
	WindowName string `json:"windowName"`
	WindowID   string `json:"windowID"`
	IsLoading  bool   `json:"visible"`
	IsActive   bool   `json:"active"`
}

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
			item.ID,
			item.Title,
			item.URL,
			item.WindowName,
			item.WindowID,
			item.IsActive,
			item.IsLoading,
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
		url := "edge://newtab/"
		if rawURL, ok := row[2].(string); ok {
			url = rawURL
		}

		cmd := exec.Command("osascript", "-e", fmt.Sprintf(newTabScript, url))
		output, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("can't run osascript: %W (message: %s)\n Script: %s", err, output, fmt.Sprintf(newTabScript, url))
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
		pk := ""
		if rawString, ok := row[0].(string); ok {
			pk = rawString
		}

		url := ""
		if rawURL, ok := row[3].(string); ok {
			url = rawURL
		}

		if url != "" {
			cmd := exec.Command("osascript", "-l", "JavaScript", "-e", fmt.Sprintf(setURLScript, pk, url))
			output, err := cmd.CombinedOutput()
			if err != nil {
				return fmt.Errorf("can't run osascript: %W (message: %s)\n Script: %s", err, output, fmt.Sprintf(setURLScript, pk, url))
			}
		}
	}

	return nil

}

// A slice of primary keys to delete
func (t *tabsTable) Delete(primaryKeys []interface{}) error {
	for _, pk := range primaryKeys {
		rawString := ""
		switch val := pk.(type) {
		case int64: // SQlite might convert the string to an int if it's a number
			rawString = fmt.Sprintf("%d", val)
		case string:
			rawString = val
		}
		if rawString == "" {
			continue
		}
		cmd := exec.Command("osascript", "-l", "JavaScript", "-e", fmt.Sprintf(deleteTabScript, rawString))
		output, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("can't run osascript: %W (message: %s)\n Script: %s", err, output, fmt.Sprintf(deleteTabScript, rawString))
		}

	}
	return nil
}

// A destructor to clean up resources
func (t *tabsTable) Close() error {
	return nil
}
