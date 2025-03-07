package main

import (
	"fmt"

	"github.com/julien040/anyquery/rpc"
	"github.com/julien040/anyquery/rpc/helper"
)

// A constructor to create a new table instance
// This function is called everytime a new connection is made to the plugin
//
// It should return a new table instance, the database schema and if there is an error
func tasksCreator(args rpc.TableCreatorArgs) (rpc.Table, *rpc.DatabaseSchema, error) {
	// Get a token from the user configuration
	// token := args.UserConfig.GetString("token")
	// if token == "" {
	// 	return nil, nil, fmt.Errorf("token must be set in the plugin configuration")
	// }

	// Open a cache connection
	/* cache, err := helper.NewCache(helper.NewCacheArgs{
		Paths:         []string{"tasks", "tasks" + "_cache"},
		EncryptionKey: []byte("my_secret_key"),
	})*/

	return &tasksTable{
			client: NewAsanaClient(args.UserConfig.GetString("token")),
		}, &rpc.DatabaseSchema{
			PartialUpdate: true,
			PrimaryKey:    1,
			Columns: []rpc.DatabaseSchemaColumn{
				{
					Name:        "project_id",
					Type:        rpc.ColumnTypeString,
					Description: "The ID of the project",
					IsParameter: true,
				},
				{
					Name:        "gid",
					Type:        rpc.ColumnTypeString,
					Description: "Globally unique task identifier.",
				},
				{
					Name:        "name",
					Type:        rpc.ColumnTypeString,
					Description: "The name of the task.",
				},
				{
					Name:        "completed",
					Type:        rpc.ColumnTypeBool,
					Description: "Task completion status.",
				},
				{
					Name:        "assignee",
					Type:        rpc.ColumnTypeString,
					Description: "Identifier of the assigned user.",
				},
				{
					Name:        "due_at",
					Type:        rpc.ColumnTypeString,
					Description: "Task due date in YYYY-MM-DD format (or YYYY-MM-DDTHH:MM:SS format).",
				},
				{
					Name:        "notes",
					Type:        rpc.ColumnTypeString,
					Description: "Task description or notes.",
				},
				{
					Name:        "created_at",
					Type:        rpc.ColumnTypeString,
					Description: "Timestamp when task was created (RFC3339 format).",
				},
				{
					Name:        "modified_at",
					Type:        rpc.ColumnTypeString,
					Description: "Timestamp when task was last modified (RFC3339 format).",
				},
				{
					Name:        "liked",
					Type:        rpc.ColumnTypeBool,
					Description: "If the task was liked by the user.",
				},
				{
					Name:        "start_at",
					Type:        rpc.ColumnTypeString,
					Description: "Task start date in YYYY-MM-DD format.",
				},
				{
					Name:        "parent",
					Type:        rpc.ColumnTypeString,
					Description: "Parent task GID if the task is a subtask.",
				},
				{
					Name:        "project",
					Type:        rpc.ColumnTypeString,
					Description: "The project name the task belongs to.",
				},
				{
					Name:        "section",
					Type:        rpc.ColumnTypeString,
					Description: "The section name the task belongs to (represents often the status of the task, such as To do, Doing, Done).",
				},
				{
					Name:        "task_type",
					Type:        rpc.ColumnTypeString,
					Description: "The type of the task (one of task, milestone, approval).",
				},
				{
					Name:        "custom_fields",
					Type:        rpc.ColumnTypeString,
					Description: "A JSON object containing the custom fields of the task (e.g. `{\"Estimated time\": \"2 hours\"}`).",
				},
			},
		}, nil
}

type tasksTable struct {
	client *AsanaClient
}

type tasksCursor struct {
	table  *tasksTable
	offset string
}

func (t *tasksTable) CreateReader() rpc.ReaderInterface {
	return &tasksCursor{
		table:  t,
		offset: "",
	}
}

func (tc *tasksCursor) Query(constraints rpc.QueryConstraint) ([][]interface{}, bool, error) {
	// Extract the project ID constraint. We assume the constraint is provided with key "project".
	projectID := constraints.GetColumnConstraint(0).GetStringValue()
	if projectID == "" {
		return nil, true, fmt.Errorf("project id constraint must be provided")
	}

	// Prepare query parameters for paging: 100 tasks at a time.
	params := map[string]string{
		"project":    projectID,
		"limit":      "100",
		"opt_fields": "resource_subtype,gid,name,completed,assignee,due_on,due_at,notes,created_at,modified_at,liked,start_at,approval_status,memberships.project.name, memberships.section.name,custom_fields.name,custom_fields.display_value",
	}
	if tc.offset != "" {
		params["offset"] = tc.offset
	}

	// Call the Asana API to get tasks for the given project.
	resp, err := tc.table.client.client.R().
		SetQueryParams(params).
		SetResult(&TasksQueryResponse{}).
		Get("/tasks")
	if err != nil {
		return nil, true, err
	}

	result := resp.Result().(*TasksQueryResponse)

	// Map tasks to rows according to the schema: [gid, name, completed, assignee, due_at, notes, created_at, modified_at].
	rows := make([][]interface{}, len(result.Data))
	for i, task := range result.Data {
		assigneeID := interface{}(nil)
		if task.Assignee != nil {
			assigneeID = task.Assignee.Gid
		}
		// Set DueAt to DueOn if DueAt is empty
		// because sometimes the API returns DueOn instead of DueAt
		if task.DueAt == "" && task.DueOn != "" {
			task.DueAt = task.DueOn
		}

		if task.StartAt == "" && task.StartOn != "" {
			task.StartAt = task.StartOn
		}

		membershipProject := interface{}(nil)
		membershipSection := interface{}(nil)
		if len(task.Memberships) > 0 {
			membershipProject = task.Memberships[0].Project.Name
			membershipSection = task.Memberships[0].Section.Name
		}

		customFields := make(map[string]interface{})
		for _, field := range task.CustomFields {
			if field.DisplayValue != nil {
				customFields[field.Name] = *field.DisplayValue
			} else {
				customFields[field.Name] = nil
			}
		}

		if task.ResourceSubtype == "default_task" {
			task.ResourceSubtype = "task"
		}

		rows[i] = []interface{}{
			task.Gid,
			task.Name,
			task.Completed,
			assigneeID,
			task.DueAt,
			task.Notes,
			task.CreatedAt,
			task.UpdatedAt,
			task.Liked,
			task.StartAt,
			task.Parent,
			membershipProject,
			membershipSection,
			task.ResourceSubtype,
			helper.Serialize(customFields),
		}
	}

	// Save the next offset token. If it's empty, we're done paging.
	tc.offset = result.NextPage.Offset
	finished := (tc.offset == "")
	return rows, finished, nil
}

// Insert creates new tasks in Asana.
func (t *tasksTable) Insert(rows [][]interface{}) error {
	// For each row, map columns to task fields and create a new task using the Asana client.
	for _, row := range rows {
		if len(row) < 11 {
			return fmt.Errorf("insert row must have 11 values: [project_id, gid, name, completed, assignee, due_at, notes, created_at, modified_at, liked, start_at]")
		}
		// Ensure project_id is not empty
		if row[0] == nil {
			return fmt.Errorf("project_id is required and cannot be empty")
		}
		projectID, ok := row[0].(string)
		if !ok || projectID == "" {
			return fmt.Errorf("project_id must be a non-empty string")
		}
		// row[1] is gid (ignored for insert)
		if row[2] == nil {
			return fmt.Errorf("name is required and cannot be nil")
		}
		name, ok := row[2].(string)
		if !ok {
			return fmt.Errorf("name must be a string")
		}
		var completed bool
		if row[3] != nil {
			completed, ok = row[3].(bool)
			if !ok {
				return fmt.Errorf("completed must be a bool if provided")
			}
		} else {
			completed = false
		}
		var dueOn string
		if row[5] != nil {
			dueOn, ok = row[5].(string)
			if !ok {
				return fmt.Errorf("due_at must be a string if provided")
			}
		} else {
			dueOn = ""
		}
		var notes string
		if row[6] != nil {
			notes, ok = row[6].(string)
			if !ok {
				return fmt.Errorf("notes must be a string if provided")
			}
		} else {
			notes = ""
		}

		var startOn string
		if row[10] != nil {
			startOn, ok = row[10].(string)
			if !ok {
				return fmt.Errorf("start_at must be a string if provided")
			}
		} else {
			startOn = ""
		}
		task := Task{
			Name:      name,
			Notes:     notes,
			DueAt:     dueOn,
			Completed: completed,
			Project:   projectID,
			StartAt:   startOn,
		}
		_, err := t.client.CreateTask(task)
		if err != nil {
			return err
		}
	}
	return nil
}

// Update modifies existing tasks in Asana.
func (t *tasksTable) Update(rows [][]interface{}) error {
	for _, row := range rows {
		if len(row) < 11 {
			return fmt.Errorf("update row must have 11 values: [old gid, project_id, new gid, name, completed, assignee, due_at, notes, created_at, modified_at, start_at]")
		}
		if row[0] == nil {
			return fmt.Errorf("old gid (primary key) is required for update and cannot be nil")
		}
		oldGid, ok := row[0].(string)
		if !ok {
			return fmt.Errorf("old gid must be a string")
		}
		var name string
		if row[3] != nil {
			name, ok = row[3].(string)
			if !ok {
				return fmt.Errorf("name must be a string")
			}
		}
		var completed bool
		if row[4] != nil {
			completedRaw, ok := row[4].(int64)
			if !ok {
				return fmt.Errorf("completed must be an int65 if provided")
			}
			completed = completedRaw == 1
		} else {
			completed = false
		}
		var dueOn string
		if row[6] != nil {
			dueOn, ok = row[6].(string)
			if !ok {
				return fmt.Errorf("due_at must be a string if provided")
			}
		} else {
			dueOn = ""
		}
		var notes string
		if row[7] != nil {
			notes, ok = row[7].(string)
			if !ok {
				return fmt.Errorf("notes must be a string if provided")
			}
		} else {
			notes = ""
		}

		var startOn string
		if row[11] != nil {
			startOn, ok = row[11].(string)
			if !ok {
				return fmt.Errorf("start_at must be a string if provided")
			}
		} else {
			startOn = ""
		}

		task := Task{
			Name:      name,
			Notes:     notes,
			DueAt:     dueOn,
			Completed: completed,
			StartAt:   startOn,
		}
		_, err := t.client.UpdateTask(oldGid, task)
		if err != nil {
			return err
		}
	}
	return nil
}

// Delete removes tasks in Asana based on their global ID.
func (t *tasksTable) Delete(primaryKeys []interface{}) error {
	for _, key := range primaryKeys {
		if key == nil {
			return fmt.Errorf("primary key (gid) cannot be nil")
		}
		id, ok := key.(string)
		if !ok {
			return fmt.Errorf("primary key must be a string")
		}
		err := t.client.DeleteTask(id)
		if err != nil {
			return err
		}
	}
	return nil
}

func (t *tasksTable) Close() error {
	return nil
}
