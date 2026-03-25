package main

import (
	"fmt"

	"github.com/julien040/anyquery/rpc"
	"github.com/julien040/anyquery/rpc/helper"
)

func tasksCreator(args rpc.TableCreatorArgs) (rpc.Table, *rpc.DatabaseSchema, error) {
	token := args.UserConfig.GetString("token")
	if token == "" {
		return nil, nil, fmt.Errorf("token must be set in the plugin configuration")
	}

	return &tasksTable{
		client: NewAsanaClient(token),
	}, &rpc.DatabaseSchema{
		PartialUpdate: true,
		PrimaryKey:    1,
		HandlesInsert: true,
		HandlesUpdate: true,
		HandlesDelete: true,
		Columns: []rpc.DatabaseSchemaColumn{
			{
				Name:        "project_id",
				Type:        rpc.ColumnTypeString,
				Description: "The GID of the project to query tasks from.",
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
				Description: "GID of the assigned user.",
			},
			{
				Name:        "due_at",
				Type:        rpc.ColumnTypeString,
				Description: "Task due date (YYYY-MM-DD or YYYY-MM-DDTHH:MM:SS).",
			},
			{
				Name:        "notes",
				Type:        rpc.ColumnTypeString,
				Description: "Task description or notes.",
			},
			{
				Name:        "created_at",
				Type:        rpc.ColumnTypeString,
				Description: "Timestamp when task was created (RFC3339).",
			},
			{
				Name:        "modified_at",
				Type:        rpc.ColumnTypeString,
				Description: "Timestamp when task was last modified (RFC3339).",
			},
			{
				Name:        "liked",
				Type:        rpc.ColumnTypeBool,
				Description: "Whether the task was liked by the current user.",
			},
			{
				Name:        "start_at",
				Type:        rpc.ColumnTypeString,
				Description: "Task start date (YYYY-MM-DD).",
			},
			{
				Name:        "parent",
				Type:        rpc.ColumnTypeString,
				Description: "Parent task GID if this is a subtask.",
			},
			{
				Name:        "project",
				Type:        rpc.ColumnTypeString,
				Description: "The project name the task belongs to.",
			},
			{
				Name:        "section",
				Type:        rpc.ColumnTypeString,
				Description: "The section name the task belongs to (e.g. To do, In progress, Done).",
			},
			{
				Name:        "task_type",
				Type:        rpc.ColumnTypeString,
				Description: "The type of the task (task, milestone, or approval).",
			},
			{
				Name:        "custom_fields",
				Type:        rpc.ColumnTypeString,
				Description: `A JSON object of custom fields (e.g. {"Estimated time": "2 hours"}).`,
			},
		},
	}, nil
}

// Schema column indices (for reference when reading Insert/Update rows):
// 0: project_id (parameter)
// 1: gid (PK)
// 2: name
// 3: completed
// 4: assignee
// 5: due_at
// 6: notes
// 7: created_at
// 8: modified_at
// 9: liked
// 10: start_at
// 11: parent
// 12: project
// 13: section
// 14: task_type
// 15: custom_fields

type tasksTable struct {
	client *AsanaClient
}

type tasksCursor struct {
	table  *tasksTable
	offset string
}

func (t *tasksTable) CreateReader() rpc.ReaderInterface {
	return &tasksCursor{table: t}
}

func (tc *tasksCursor) Query(constraints rpc.QueryConstraint) ([][]interface{}, bool, error) {
	projectID := constraints.GetColumnConstraint(0).GetStringValue()
	if projectID == "" {
		return nil, true, fmt.Errorf("project_id constraint must be provided")
	}

	params := map[string]string{
		"project":    projectID,
		"limit":      "100",
		"opt_fields": "resource_subtype,gid,name,completed,assignee.gid,due_on,due_at,notes,created_at,modified_at,liked,start_at,start_on,approval_status,memberships.project.name,memberships.section.name,custom_fields.name,custom_fields.display_value",
	}
	if tc.offset != "" {
		params["offset"] = tc.offset
	}

	resp, err := tc.table.client.client.R().
		SetQueryParams(params).
		SetResult(&TasksQueryResponse{}).
		Get("/tasks")
	if err != nil {
		return nil, true, err
	}
	if resp.IsError() {
		return nil, true, fmt.Errorf("failed to fetch tasks (%d): %s", resp.StatusCode(), resp.String())
	}

	result, ok := resp.Result().(*TasksQueryResponse)
	if !ok || result == nil {
		return nil, true, fmt.Errorf("unexpected response format")
	}

	rows := make([][]interface{}, len(result.Data))
	for i, task := range result.Data {
		var assigneeGid interface{}
		if task.Assignee != nil {
			assigneeGid = task.Assignee.Gid
		}

		// Prefer DueAt; fall back to DueOn
		dueAt := task.DueAt
		if dueAt == "" {
			dueAt = task.DueOn
		}

		// Prefer StartAt; fall back to StartOn
		startAt := task.StartAt
		if startAt == "" {
			startAt = task.StartOn
		}

		var membershipProject, membershipSection interface{}
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

		taskType := task.ResourceSubtype
		if taskType == "default_task" {
			taskType = "task"
		}

		rows[i] = []interface{}{
			task.Gid,
			task.Name,
			task.Completed,
			assigneeGid,
			nilIfEmpty(dueAt),
			nilIfEmpty(task.Notes),
			task.CreatedAt,
			task.UpdatedAt,
			task.Liked,
			nilIfEmpty(startAt),
			nilIfEmpty(task.Parent),
			membershipProject,
			membershipSection,
			nilIfEmpty(taskType),
			helper.Serialize(customFields),
		}
	}

	tc.offset = result.NextPage.Offset
	return rows, tc.offset == "", nil
}

// Insert creates new tasks in Asana.
// Row layout: [project_id, gid, name, completed, assignee, due_at, notes, created_at, modified_at, liked, start_at, parent, project, section, task_type, custom_fields]
func (t *tasksTable) Insert(rows [][]interface{}) error {
	for _, row := range rows {
		if len(row) < 3 {
			return fmt.Errorf("insert row too short")
		}

		projectID, _ := row[0].(string)
		if projectID == "" {
			return fmt.Errorf("project_id is required")
		}

		name, _ := row[2].(string)
		if name == "" {
			return fmt.Errorf("name is required")
		}

		task := Task{
			Name:    name,
			Project: projectID,
		}

		if len(row) > 5 && row[5] != nil {
			task.DueAt, _ = row[5].(string)
		}
		if len(row) > 6 && row[6] != nil {
			task.Notes, _ = row[6].(string)
		}
		if len(row) > 10 && row[10] != nil {
			task.StartAt, _ = row[10].(string)
		}
		if len(row) > 3 && row[3] != nil {
			task.Completed = asBool(row[3])
		}

		if _, err := t.client.CreateTask(task); err != nil {
			return err
		}
	}
	return nil
}

// Update modifies existing tasks in Asana.
// Row layout: [old_gid, project_id, new_gid, name, completed, assignee, due_at, notes, created_at, modified_at, liked, start_at, ...]
func (t *tasksTable) Update(rows [][]interface{}) error {
	for _, row := range rows {
		if len(row) < 3 {
			return fmt.Errorf("update row too short")
		}

		oldGid, ok := row[0].(string)
		if !ok || oldGid == "" {
			return fmt.Errorf("primary key (gid) is required for update")
		}

		data := make(map[string]interface{})

		if len(row) > 3 && row[3] != nil {
			if name, ok := row[3].(string); ok && name != "" {
				data["name"] = name
			}
		}
		if len(row) > 4 && row[4] != nil {
			data["completed"] = asBool(row[4])
		}
		if len(row) > 6 && row[6] != nil {
			if due, ok := row[6].(string); ok {
				data["due_on"] = due
			}
		}
		if len(row) > 7 && row[7] != nil {
			if notes, ok := row[7].(string); ok {
				data["notes"] = notes
			}
		}
		if len(row) > 11 && row[11] != nil {
			if start, ok := row[11].(string); ok {
				data["start_on"] = start
			}
		}

		if len(data) == 0 {
			continue
		}

		if _, err := t.client.UpdateTask(oldGid, data); err != nil {
			return err
		}
	}
	return nil
}

// Delete removes tasks in Asana.
func (t *tasksTable) Delete(primaryKeys []interface{}) error {
	for _, key := range primaryKeys {
		if key == nil {
			return fmt.Errorf("primary key (gid) cannot be nil")
		}
		id, ok := key.(string)
		if !ok || id == "" {
			return fmt.Errorf("primary key must be a non-empty string")
		}
		if err := t.client.DeleteTask(id); err != nil {
			return err
		}
	}
	return nil
}

func (t *tasksTable) Close() error {
	return nil
}

// asBool converts an interface{} value (bool or int64) to bool.
func asBool(v interface{}) bool {
	switch val := v.(type) {
	case bool:
		return val
	case int64:
		return val != 0
	}
	return false
}

// nilIfEmpty returns nil if s is empty, otherwise s.
func nilIfEmpty(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}
