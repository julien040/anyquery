package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/hashicorp/go-retryablehttp"
	"github.com/julien040/anyquery/rpc"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/option"
	"google.golang.org/api/tasks/v1"
)

// A constructor to create a new table instance
// This function is called everytime a new connection is made to the plugin
//
// It should return a new table instance, the database schema and if there is an error
func tasksCreator(args rpc.TableCreatorArgs) (rpc.Table, *rpc.DatabaseSchema, error) {
	var token, clientID, clientSecret string

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

	config := oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		Endpoint:     google.Endpoint,
		Scopes:       []string{tasks.TasksScope},
	}

	oauthClient := config.Client(context.Background(), &oauth2.Token{
		RefreshToken: token,
	})

	retryableClient := retryablehttp.NewClient()
	retryableClient.HTTPClient = oauthClient

	srv, err := tasks.NewService(context.Background(), option.WithHTTPClient(retryableClient.StandardClient()))
	if err != nil {
		return nil, nil, fmt.Errorf("unable to create people service: %w", err)
	}
	return &tasksTable{
			srv: srv,
		}, &rpc.DatabaseSchema{
			HandlesInsert: true,
			HandlesUpdate: true,
			// Due to the way Google Tasks API works, we can't delete tasks
			// It's because anyquery only passes the PrimaryKey to the Delete function
			// while we need the pks and the list_id to delete a task
			HandlesDelete: false,
			PrimaryKey:    2,
			Columns: []rpc.DatabaseSchemaColumn{
				{
					Name:        "list_id",
					Type:        rpc.ColumnTypeString,
					IsParameter: true,
					IsRequired:  true,
				},
				{
					Name:        "show_deleted",
					Type:        rpc.ColumnTypeInt,
					IsParameter: true,
				},
				{
					Name: "id",
					Type: rpc.ColumnTypeString,
				},
				{
					Name: "title",
					Type: rpc.ColumnTypeString,
				},
				{
					Name: "status",
					Type: rpc.ColumnTypeString,
				},
				{
					Name: "completed_at",
					Type: rpc.ColumnTypeString,
				},
				{
					Name: "due_at",
					Type: rpc.ColumnTypeString,
				},
				{
					Name: "updated_at",
					Type: rpc.ColumnTypeString,
				},
				{
					Name: "links",
					Type: rpc.ColumnTypeString,
				},
				{
					Name: "notes",
					Type: rpc.ColumnTypeString,
				},
				{
					Name: "parent_id",
					Type: rpc.ColumnTypeString,
				},
				{
					Name: "position",
					Type: rpc.ColumnTypeString,
				},
				{
					Name: "url",
					Type: rpc.ColumnTypeString,
				},
				{
					Name: "hidden",
					Type: rpc.ColumnTypeBool,
				},
				{
					Name: "deleted",
					Type: rpc.ColumnTypeBool,
				},
			},
		}, nil
}

type tasksTable struct {
	srv *tasks.Service
}

type tasksCursor struct {
	srv        *tasks.Service
	nextCursor string
}

// Return a slice of rows that will be returned to Anyquery and filtered.
// The second return value is true if the cursor has no more rows to return
//
// The constraints are used for optimization purposes to "pre-filter" the rows
// If the rows returned don't match the constraints, it's not an issue. Anyquery will filter them out
func (t *tasksCursor) Query(constraints rpc.QueryConstraint) ([][]interface{}, bool, error) {
	// Retrieve the task list
	listID := ""
	deleted := false
	for _, c := range constraints.Columns {
		if c.ColumnID == 0 {
			var ok bool
			if listID, ok = c.Value.(string); !ok {
				return nil, true, fmt.Errorf("list_id should be a string")
			}
		}
		if c.ColumnID == 1 {
			log.Printf("Value: %v %T", c.Value, c.Value)
			if val, ok := c.Value.(int64); ok {
				deleted = val == 1
			}
		}
	}
	if listID == "" {
		return nil, true, fmt.Errorf("list_id is required")
	}

	log.Printf("Will show deleted: %v", deleted)

	req := t.srv.Tasks.List(listID).ShowAssigned(true).
		ShowCompleted(true).ShowDeleted(deleted).ShowHidden(true).MaxResults(100)
	if t.nextCursor != "" {
		req = req.PageToken(t.nextCursor)
	}

	log.Printf("Querying tasks with next cursor: %s", t.nextCursor)
	tasks, err := req.Do()
	if err != nil {
		return nil, true, fmt.Errorf("unable to list tasks: %w", err)
	}

	rows := make([][]interface{}, 0, len(tasks.Items))
	for _, task := range tasks.Items {
		completed := interface{}(nil)
		if task.Completed != nil {
			completed = task.Completed
		}
		var links []string
		for _, link := range task.Links {
			links = append(links, link.Link)
		}
		rows = append(rows, []interface{}{
			task.Id,
			task.Title,
			task.Status,
			completed,
			task.Due,
			task.Updated,
			links,
			task.Notes,
			task.Parent,
			task.Position,
			task.WebViewLink,
			task.Deleted,
			task.Hidden,
			task.Deleted,
		})
	}

	log.Printf("Returning %d tasks and got next cursor: %s", len(rows), tasks.NextPageToken)
	t.nextCursor = tasks.NextPageToken

	return rows, tasks.NextPageToken == "", nil
}

// Create a new cursor that will be used to read rows
func (t *tasksTable) CreateReader() rpc.ReaderInterface {
	return &tasksCursor{
		srv: t.srv,
	}
}

// A slice of rows to insert
func (t *tasksTable) Insert(rows [][]interface{}) error {
	for i, row := range rows {
		// Extract the list ID
		listID := extractString(row, 0)
		if listID == "" {
			return fmt.Errorf("list_id is required. Specify it like this: INSERT INTO tasks (list_id, ...) VALUES ('list_id', ...)")
		}

		// Extract the informations
		title := extractString(row, 3)
		status := extractString(row, 4)
		if status != "needsAction" && status != "completed" {
			status = "needsAction"
		}
		completed := extractString(row, 5)
		due := extractString(row, 6)
		parsedDue, err := parseTime(due)
		if err == nil {
			due = parsedDue.Format(time.RFC3339)
		} else {
			due = ""
		}

		notes := extractString(row, 9)

		if completed == "" && status == "completed" {
			completed = time.Now().Format(time.RFC3339)
		} else if completed != "" && status != "completed" {
			parsedCompleted, err := parseTime(completed)
			if err == nil {
				completed = parsedCompleted.Format(time.RFC3339)
				status = "completed"
			} else {
				completed = ""
			}
		}
		task := &tasks.Task{
			Title:  title,
			Status: status,
			Due:    due,
			Notes:  notes,
		}

		if completed != "" {
			task.Completed = &completed
		}

		// Insert the task
		_, err = t.srv.Tasks.Insert(listID, task).Do()
		if err != nil {
			return fmt.Errorf("unable to insert task(index: %d): %w", i, err)
		}
	}

	return nil
}

// A slice of rows to update
// The first element of each row is the primary key
// while the rest are the values to update
// The primary key is therefore present twice
func (t *tasksTable) Update(rows [][]interface{}) error {
	for i, row := range rows {
		// The first column is the primary key
		taskID := extractString(row, 0)
		// Extract the list ID
		listID := extractString(row, 1)

		title := extractString(row, 4)
		status := extractString(row, 5)
		if status != "needsAction" && status != "completed" {
			status = "needsAction"
		}
		completed := extractString(row, 6)
		if completed != "" {
			parsedCompleted, err := parseTime(completed)
			if err == nil {
				completed = parsedCompleted.Format(time.RFC3339)
			} else {
				completed = ""
			}
		}
		due := extractString(row, 7)
		parsedDue, err := parseTime(due)
		if err == nil {
			due = parsedDue.Format(time.RFC3339)
		} else {
			due = ""
		}

		notes := extractString(row, 10)

		if completed == "" && status == "completed" {
			completed = time.Now().Format(time.RFC3339)
		}
		task := &tasks.Task{
			Title:  title,
			Status: status,
			Due:    due,
			Id:     taskID,
			Notes:  notes,
		}

		if completed != "" {
			task.Completed = &completed
		}

		log.Printf("Updating task: %s of list: %s", taskID, listID)
		// Update the task
		_, err = t.srv.Tasks.Update(listID, taskID, task).Do()
		if err != nil {
			return fmt.Errorf("unable to update task(index: %d): %w", i, err)
		}
	}

	return nil
}

// A slice of primary keys to delete
func (t *tasksTable) Delete(primaryKeys []interface{}) error {
	return fmt.Errorf("deleting tasks is not supported")
}

// A destructor to clean up resources
func (t *tasksTable) Close() error {
	return nil
}

func extractString(row []interface{}, index int) string {
	if index >= len(row) || index < 0 {
		return ""
	}

	if row[index] == nil {
		return ""
	}

	if val, ok := row[index].(string); ok {
		return val
	}
	return ""
}

func parseTime(timeStr string) (time.Time, error) {
	timeAllowedFormats := []string{
		time.RFC3339,
		time.DateOnly,
		"01/02/2006",
	}

	for _, format := range timeAllowedFormats {
		if t, err := time.Parse(format, timeStr); err == nil {
			return t, nil
		}
	}

	return time.Time{}, fmt.Errorf("unable to parse time: %s", timeStr)
}
