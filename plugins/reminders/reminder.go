//go:build darwin

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
	"sync"
	"text/template"
	"time"

	"github.com/julien040/anyquery/rpc"

	_ "embed"
)

//go:embed script/list.applescript
var listAppleScript string

//go:embed script/create.applescript
var createAppleScript string
var createTemplate = template.Must(template.New("create").Parse(createAppleScript))

//go:embed script/update.applescript
var updateAppleScript string
var updateTemplate = template.Must(template.New("update").Parse(updateAppleScript))

type templateArg struct {
	ID        string
	Name      string
	Body      string
	Completed string
	DueDate   string
	Priority  string
	Day       string
	Month     string
	Year      string
	Minute    string
	Hour      string
	List      string
}

//go:embed script/delete.applescript
var deleteAppleScript string
var deleteTemplate = template.Must(template.New("delete").Parse(deleteAppleScript))

type itemList struct {
	ID        string  `json:"id"`
	Name      string  `json:"name"`
	Body      *string `json:"body"`
	Completed bool    `json:"completed"`
	DueDate   *string `json:"due_date"`
	Priority  int     `json:"priority"`
	List      string  `json:"list"`
}

// A constructor to create a new table instance
// This function is called everytime a new connection is made to the plugin
//
// It should return a new table instance, the database schema and if there is an error
func reminderCreator(args rpc.TableCreatorArgs) (rpc.Table, *rpc.DatabaseSchema, error) {
	rowCache := [][]interface{}{}
	rowCacheMutex := &sync.Mutex{}
	return &reminderTable{
			rowCache:      rowCache,
			rowCacheMutex: rowCacheMutex,
		}, &rpc.DatabaseSchema{
			HandlesInsert: true,
			HandlesUpdate: true,
			HandlesDelete: true,
			Columns: []rpc.DatabaseSchemaColumn{
				{
					Name:        "id",
					Type:        rpc.ColumnTypeString,
					Description: "The ID of the reminder",
				},
				{
					Name:        "list",
					Type:        rpc.ColumnTypeString,
					Description: "The name of the list that the reminder belongs to",
				},
				{
					Name:        "name",
					Type:        rpc.ColumnTypeString,
					Description: "The name of the reminder",
				},
				{
					Name:        "body",
					Type:        rpc.ColumnTypeString,
					Description: "The body of the reminder. Can contain additional information about the reminder",
				},
				{
					Name:        "completed",
					Type:        rpc.ColumnTypeBool,
					Description: "Whether the reminder is completed or not",
				},
				{
					Name:        "due_date",
					Type:        rpc.ColumnTypeDateTime,
					Description: "The due date of the reminder, if any",
				},
				{
					Name:        "priority",
					Type:        rpc.ColumnTypeInt,
					Description: "The priority of the reminder",
				},
			},
		}, nil
}

type reminderTable struct {
	rowCache      [][]interface{}
	rowCacheMutex *sync.Mutex
	decoder       *json.Decoder
	decoderIndex  int
}

type reminderCursor struct {
	currentRow      int
	rowCache        *[][]interface{}
	rowCacheMutex   *sync.Mutex
	decoder         **json.Decoder
	decoderIndex    *int
	cursorExhausted bool
}

// Return a slice of rows that will be returned to Anyquery and filtered.
// The second return value is true if the cursor has no more rows to return
//
// The constraints are used for optimization purposes to "pre-filter" the rows
// If the rows returned don't match the constraints, it's not an issue. Anyquery will filter them out
func (t *reminderCursor) Query(constraints rpc.QueryConstraint) ([][]interface{}, bool, error) {
	if t.cursorExhausted {
		return nil, true, nil
	}

	// Check if the row is in the cache
	if t.currentRow < len(*t.rowCache) {
		row := (*t.rowCache)[t.currentRow]
		t.currentRow++
		return [][]interface{}{row}, false, nil
	}

	// Fetch the next row from the database
	if *t.decoder == nil {
		stdout, _, err := runOsaScript(listAppleScript)
		if err != nil {
			return nil, true, fmt.Errorf("failed to run osascript: %w", err)
		}

		*t.decoder = json.NewDecoder(stdout)
	}

	for t.currentRow >= *t.decoderIndex {
		row := itemList{}
		err := (*t.decoder).Decode(&row)
		if err == io.EOF {
			t.cursorExhausted = true
			break
		} else if err != nil {
			return nil, true, fmt.Errorf("failed to decode JSON: %w", err)
		}
		body := interface{}("")
		date := interface{}("")
		if row.Body != nil {
			body = *row.Body
		}
		if row.DueDate != nil {
			// parse the date as YYYY-MM-DD
			parsed, err := time.Parse("2006-01-02", *row.DueDate)
			if err == nil {
				date = parsed.Format(time.RFC3339)
			}
		}

		t.rowCacheMutex.Lock()
		*t.rowCache = append(*t.rowCache, []interface{}{
			row.ID,
			row.List,
			row.Name,
			body,
			row.Completed,
			date,
			row.Priority,
		})

		t.rowCacheMutex.Unlock()
		*t.decoderIndex++
	}

	if t.currentRow < len(*t.rowCache) {
		row := (*t.rowCache)[t.currentRow]
		t.currentRow++
		return [][]interface{}{row}, false, nil
	} else {
		t.cursorExhausted = true
		return nil, true, nil
	}

}

// Create a new cursor that will be used to read rows
func (t *reminderTable) CreateReader() rpc.ReaderInterface {
	return &reminderCursor{
		currentRow:    0,
		rowCache:      &t.rowCache,
		rowCacheMutex: t.rowCacheMutex,
		decoder:       &t.decoder,
		decoderIndex:  &t.decoderIndex,
	}
}

// A slice of rows to insert
func (t *reminderTable) Insert(rows [][]interface{}) error {
	defer clear(t.rowCache)
	for _, row := range rows {
		createArgs := templateArg{}
		if row[1] != nil {
			createArgs.List = row[1].(string)
		}
		if row[2] != nil {
			createArgs.Name = row[2].(string)
		}
		if row[3] != nil {
			createArgs.Body = row[3].(string)
		}
		if row[4] != nil {
			boolVal := row[4].(int64)
			if boolVal == 1 {
				createArgs.Completed = "true"
			} else {
				createArgs.Completed = "false"
			}
		}
		if row[5] != nil {
			date := row[5].(string)
			parsedDate, err := time.Parse("2006-01-02 15:04", date)
			if err != nil {
				parsedDate, err = time.Parse(time.RFC3339, date)
			}
			if err == nil {
				createArgs.Day = strconv.Itoa(parsedDate.Day())
				createArgs.Month = strconv.Itoa(int(parsedDate.Month()))
				createArgs.Year = strconv.Itoa(parsedDate.Year())
				createArgs.Minute = strconv.Itoa(parsedDate.Minute())
				createArgs.Hour = strconv.Itoa(parsedDate.Hour())
			} else {
				// Parse the date as YYYY-MM-DD
				parsedDate, err := time.Parse("2006-01-02", date)
				if err == nil {
					createArgs.Day = strconv.Itoa(parsedDate.Day())
					createArgs.Month = strconv.Itoa(int(parsedDate.Month()))
					createArgs.Year = strconv.Itoa(parsedDate.Year())
				}

			}
		}
		if row[6] != nil {
			createArgs.Priority = strconv.Itoa(int(row[6].(int64)))
		}

		if createArgs.List == "" {
			return fmt.Errorf("list is required")
		}

		templateRun := &bytes.Buffer{}
		if err := createTemplate.Execute(templateRun, createArgs); err != nil {
			return fmt.Errorf("failed to execute template: %w", err)
		}

		stdout, cmd, err := runOsaScript(templateRun.String())
		if err != nil {
			return fmt.Errorf("failed to run osascript: %w", err)
		}

		if err := cmd.Wait(); err != nil {
			return fmt.Errorf("failed to wait for command: %w (%s)", err, stdout)
		}
	}
	return nil
}

// A slice of rows to update
// The first element of each row is the primary key
// while the rest are the values to update
// The primary key is therefore present twice
func (t *reminderTable) Update(rows [][]interface{}) error {
	defer clear(t.rowCache)
	for _, row := range rows {
		updateArgs := templateArg{}
		updateArgs.ID = row[0].(string)
		if row[3] != nil {
			updateArgs.Name = row[3].(string)
		}
		if row[4] != nil {
			updateArgs.Body = row[4].(string)
		}
		if row[5] != nil {
			boolVal := row[5].(int64)
			if boolVal == 1 {
				updateArgs.Completed = "true"
			} else {
				updateArgs.Completed = "false"
			}
		}
		if row[6] != nil {
			date := row[6].(string)
			parsedDate, err := time.Parse("2006-01-02 15:04", date)
			if err != nil {
				parsedDate, err = time.Parse(time.RFC3339, date)
			}
			if err == nil {
				updateArgs.Day = strconv.Itoa(parsedDate.Day())
				updateArgs.Month = strconv.Itoa(int(parsedDate.Month()))
				updateArgs.Year = strconv.Itoa(parsedDate.Year())
				updateArgs.Minute = strconv.Itoa(parsedDate.Minute())
				updateArgs.Hour = strconv.Itoa(parsedDate.Hour())
			} else {
				// Parse the date as YYYY-MM-DD
				parsedDate, err := time.Parse("2006-01-02", date)
				if err == nil {
					updateArgs.Day = strconv.Itoa(parsedDate.Day())
					updateArgs.Month = strconv.Itoa(int(parsedDate.Month()))
					updateArgs.Year = strconv.Itoa(parsedDate.Year())
				}

			}
		}
		if row[7] != nil {
			updateArgs.Priority = strconv.Itoa(int(row[7].(int64)))
		}

		templateRun := &bytes.Buffer{}
		if err := updateTemplate.Execute(templateRun, updateArgs); err != nil {
			return fmt.Errorf("failed to execute template: %w", err)
		}

		stdout, cmd, err := runOsaScript(templateRun.String())
		if err != nil {
			return fmt.Errorf("failed to run osascript: %w", err)
		}

		if err := cmd.Wait(); err != nil {
			return fmt.Errorf("failed to wait for command: %w (%s)", err, stdout)
		}
	}

	return nil
}

// A slice of primary keys to delete
func (t *reminderTable) Delete(primaryKeys []interface{}) error {
	defer clear(t.rowCache)
	for _, key := range primaryKeys {
		deleteArgs := templateArg{}
		deleteArgs.ID = key.(string)

		templateRun := &bytes.Buffer{}
		if err := deleteTemplate.Execute(templateRun, deleteArgs); err != nil {
			return fmt.Errorf("failed to execute template: %w", err)
		}

		stdout, cmd, err := runOsaScript(templateRun.String())
		if err != nil {
			return fmt.Errorf("failed to run osascript: %w", err)
		}

		if err := cmd.Wait(); err != nil {
			return fmt.Errorf("failed to wait for command: %w (%s)", err, stdout)
		}
	}
	return nil
}

// A destructor to clean up resources
func (t *reminderTable) Close() error {
	return nil
}

func runOsaScript(script string) (io.Reader, *exec.Cmd, error) {
	// Create a new command
	cmd := exec.Command("osascript", "-sh", "-e", script)
	// plugin only supports printing to stderr
	cmd.Stdout = os.Stderr

	// Run the command
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, nil, fmt.Errorf("failed to start command: %w", err)
	}

	return stderr, cmd, nil
}
