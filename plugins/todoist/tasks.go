package main

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/hashicorp/go-retryablehttp"
	"github.com/julien040/anyquery/rpc"
)

var retryClient = retryablehttp.NewClient()
var client = resty.NewWithClient(retryClient.StandardClient())

// A constructor to create a new table instance
// This function is called everytime a new connection is made to the plugin
//
// It should return a new table instance, the database schema and if there is an error
func tasksCreator(args rpc.TableCreatorArgs) (rpc.Table, *rpc.DatabaseSchema, error) {
	token := ""
	if inter, ok := args.UserConfig["token"]; ok {
		var ok bool
		token, ok = inter.(string)
		if !ok {
			return nil, nil, fmt.Errorf("token is not a string")
		}
	}
	if token == "" {
		return nil, nil, fmt.Errorf("token is empty")
	}
	return &tasksTable{
			token: token,
		}, &rpc.DatabaseSchema{
			HandlesInsert: true,
			HandlesUpdate: false,
			HandlesDelete: true,
			Columns: []rpc.DatabaseSchemaColumn{
				{
					Name: "id",
					Type: rpc.ColumnTypeString,
				},
				{
					Name: "assigner_id",
					Type: rpc.ColumnTypeString,
				},
				{
					Name: "assignee_id",
					Type: rpc.ColumnTypeString,
				},
				{
					Name: "project_id",
					Type: rpc.ColumnTypeString,
				},
				{
					Name: "section_id",
					Type: rpc.ColumnTypeString,
				},
				{
					Name: "parent_id",
					Type: rpc.ColumnTypeString,
				},
				{
					Name: "order",
					Type: rpc.ColumnTypeInt,
				},
				{
					Name: "content",
					Type: rpc.ColumnTypeString,
				},
				{
					Name: "description",
					Type: rpc.ColumnTypeString,
				},
				{
					Name: "is_completed",
					Type: rpc.ColumnTypeBool,
				},
				{
					Name: "labels",
					Type: rpc.ColumnTypeString,
				},
				{
					Name: "priority",
					Type: rpc.ColumnTypeInt,
				},
				{
					Name: "comment_count",
					Type: rpc.ColumnTypeInt,
				},
				{
					Name: "creator_id",
					Type: rpc.ColumnTypeString,
				},
				{
					Name: "created_at",
					Type: rpc.ColumnTypeString,
				},
				{
					Name: "due",
					Type: rpc.ColumnTypeString,
				},
				{
					Name: "url",
					Type: rpc.ColumnTypeString,
				},
			},
		}, nil
}

type tasksTable struct {
	token string
}

type tasksCursor struct {
	token string
}

// Return a slice of rows that will be returned to Anyquery and filtered.
// The second return value is true if the cursor has no more rows to return
//
// The constraints are used for optimization purposes to "pre-filter" the rows
// If the rows returned don't match the constraints, it's not an issue. Anyquery will filter them out
func (t *tasksCursor) Query(constraints rpc.QueryConstraint) ([][]interface{}, bool, error) {
	// Retrieve the tasks
	endpoint := "https://api.todoist.com/rest/v2/tasks"
	body := &Tasks{}
	resp, err := client.R().
		SetHeader("Authorization", "Bearer "+t.token).
		SetResult(body).
		Get(endpoint)
	if err != nil {
		return nil, true, fmt.Errorf("failed to list tasks: %w", err)
	}

	if resp.IsError() {
		return nil, true, fmt.Errorf("failed to list tasks(code %d): %s", resp.StatusCode(), resp.String())
	}

	rows := [][]interface{}{}

	for _, task := range *body {
		assigner := interface{}(nil)
		if task.AssignerID != nil {
			assigner = *task.AssignerID
		}
		assignee := interface{}(nil)
		if task.AssigneeID != nil {
			assignee = *task.AssigneeID
		}
		section := interface{}(nil)
		if task.SectionID != nil {
			section = *task.SectionID
		}
		dueDate := interface{}(nil)
		if task.Due != nil {
			dueDate = task.Due.Date
		}
		rows = append(rows, []interface{}{
			task.ID,
			assigner,
			assignee,
			task.ProjectID,
			section,
			task.ParentID,
			task.Order,
			task.Content,
			task.Description,
			task.IsCompleted,
			task.Labels,
			task.Priority,
			task.CommentCount,
			task.CreatorID,
			task.CreatedAt,
			dueDate,
			task.URL,
		})
	}

	return rows, true, nil
}

// Create a new cursor that will be used to read rows
func (t *tasksTable) CreateReader() rpc.ReaderInterface {
	return &tasksCursor{
		token: t.token,
	}
}

// A slice of rows to insert
func (t *tasksTable) Insert(rows [][]interface{}) error {
	for _, row := range rows {
		task := Task{}
		if val := getString(row, 1); val != "" {
			task.AssignerID = &val
		}
		if val := getString(row, 2); val != "" {
			task.AssigneeID = &val
		}
		if val := getString(row, 3); val != "" {
			task.ProjectID = val
		}
		if val := getString(row, 4); val != "" {
			task.SectionID = &val
		}
		if val := getString(row, 5); val != "" {
			task.ParentID = val
		}
		if val := getInteger(row, 6); val > 0 {
			task.Order = val
		}
		if val := getString(row, 7); val != "" {
			task.Content = val
		} else {
			// Content is required
			task.Content = "New Task"
		}
		if val := getString(row, 8); val != "" {
			task.Description = val
		}
		if val := getString(row, 10); val != "" {
			// Try to parse it as a JSON array of strings
			labels := []string{}
			if err := json.Unmarshal([]byte(val), &labels); err == nil {
				task.Labels = labels
			}
		}
		if val := getInteger(row, 11); val > 0 && val < 5 {
			task.Priority = val
		}
		if val := getString(row, 15); val != "" {
			// Check if it's a valid date
			if _, err := time.Parse("2006-01-02", val); err == nil {
				task.Due = &Due{Date: val}
			}
		}

		// Create the task
		endpoint := "https://api.todoist.com/rest/v2/tasks"
		resp, err := client.R().
			SetHeader("Authorization", "Bearer "+t.token).
			SetHeader("Content-Type", "application/json").
			SetBody(task).
			Post(endpoint)

		if err != nil {
			return fmt.Errorf("failed to create task: %w", err)
		}

		if resp.IsError() {
			return fmt.Errorf("failed to create task(code %d): %s", resp.StatusCode(), resp.String())
		}
	}
	return nil
}

// A slice of rows to update
// The first element of each row is the primary key
// while the rest are the values to update
// The primary key is therefore present twice
func (t *tasksTable) Update(rows [][]interface{}) error {
	return nil
}

// A slice of primary keys to delete
func (t *tasksTable) Delete(primaryKeys []interface{}) error {
	for _, primaryKey := range primaryKeys {
		// Ensure the primary key is a string
		id, ok := primaryKey.(string)
		if !ok {
			return fmt.Errorf("primary key is not a string")
		}
		endpoint := fmt.Sprintf("https://api.todoist.com/rest/v2/tasks/%s/close", id)
		resp, err := client.R().
			SetHeader("Authorization", "Bearer "+t.token).
			Post(endpoint)

		if err != nil {
			return fmt.Errorf("failed to close task: %w", err)
		}

		if resp.IsError() {
			return fmt.Errorf("failed to close task(code %d): %s", resp.StatusCode(), resp.String())
		}

	}
	return nil
}

// A destructor to clean up resources
func (t *tasksTable) Close() error {
	return nil
}

func getString(row []interface{}, index int) string {
	if row[index] == nil {
		return ""
	}
	switch row[index].(type) {
	case string:
		return row[index].(string)
	case int64:
		return fmt.Sprintf("%d", row[index].(int64))
	case float64:
		return fmt.Sprintf("%f", row[index].(float64))
	case bool:
		return fmt.Sprintf("%t", row[index].(bool))
	}

	return ""
}

func getInteger(row []interface{}, index int) int64 {
	if row[index] == nil {
		return 0
	}
	switch row[index].(type) {
	case string:
		// Try to convert the string to an integer
		i, err := strconv.ParseInt(row[index].(string), 10, 64)
		if err != nil {
			return 0
		}
		return i
	case int64:
		return row[index].(int64)
	case float64:
		return int64(row[index].(float64))
	case bool:
		return 0
	}

	return 0
}
