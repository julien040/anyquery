package main

import (
	"fmt"

	"github.com/go-resty/resty/v2"
)

// NextPage holds the pagination token from Asana list responses.
type NextPage struct {
	Offset string `json:"offset"`
}

// ---- Task types ----

type TasksResponse struct {
	Data []Task `json:"data"`
}

type TaskResponse struct {
	Data Task `json:"data"`
}

type TasksQueryResponse struct {
	Data     []Task   `json:"data"`
	NextPage NextPage `json:"next_page"`
}

// ---- Project types ----

type Project struct {
	Gid        string        `json:"gid,omitempty"`
	Name       string        `json:"name,omitempty"`
	Owner      *User         `json:"owner,omitempty"`
	CreatedAt  string        `json:"created_at,omitempty"`
	ModifiedAt string        `json:"modified_at,omitempty"`
	Archived   bool          `json:"archived,omitempty"`
	Color      string        `json:"color,omitempty"`
	Notes      string        `json:"notes,omitempty"`
	Workspace  ProjectSection `json:"workspace,omitempty"`
	Team       ProjectSection `json:"team,omitempty"`
}

type ProjectsQueryResponse struct {
	Data     []Project `json:"data"`
	NextPage NextPage  `json:"next_page"`
}

// ---- Goal types ----

type Goal struct {
	Gid       string        `json:"gid,omitempty"`
	Name      string        `json:"name,omitempty"`
	Owner     *User         `json:"owner,omitempty"`
	CreatedAt string        `json:"created_at,omitempty"`
	DueOn     string        `json:"due_on,omitempty"`
	Status    string        `json:"status,omitempty"`
	Notes     string        `json:"notes,omitempty"`
	Workspace ProjectSection `json:"workspace,omitempty"`
}

type GoalsQueryResponse struct {
	Data     []Goal   `json:"data"`
	NextPage NextPage `json:"next_page"`
}

// ---- Workspace types ----

type Workspace struct {
	Gid            string `json:"gid,omitempty"`
	Name           string `json:"name,omitempty"`
	IsOrganization bool   `json:"is_organization,omitempty"`
}

type WorkspacesQueryResponse struct {
	Data     []Workspace `json:"data"`
	NextPage NextPage    `json:"next_page"`
}

// ---- Shared sub-types ----

type Memberships struct {
	Project ProjectSection `json:"project,omitempty"`
	Section ProjectSection `json:"section,omitempty"`
}

type ProjectSection struct {
	Gid  string `json:"gid,omitempty"`
	Name string `json:"name,omitempty"`
}

// User represents an Asana user.
type User struct {
	Gid  string `json:"gid"`
	Name string `json:"name"`
}

type CustomField struct {
	Gid          string  `json:"gid,omitempty"`
	Name         string  `json:"name,omitempty"`
	DisplayValue *string `json:"display_value,omitempty"`
}

// Task represents an Asana task with the most common fields.
type Task struct {
	Gid             string        `json:"gid,omitempty"`
	Name            string        `json:"name,omitempty"`
	Completed       bool          `json:"completed,omitempty"`
	Liked           bool          `json:"liked,omitempty"`
	Assignee        *User         `json:"assignee,omitempty"`
	DueOn           string        `json:"due_on,omitempty"`
	DueAt           string        `json:"due_at,omitempty"`
	StartAt         string        `json:"start_at,omitempty"`
	StartOn         string        `json:"start_on,omitempty"`
	Parent          string        `json:"parent,omitempty"`
	Notes           string        `json:"notes,omitempty"`
	Memberships     []Memberships `json:"memberships,omitempty"`
	CreatedAt       string        `json:"created_at,omitempty"`
	UpdatedAt       string        `json:"modified_at,omitempty"`
	ResourceSubtype string        `json:"resource_subtype,omitempty"`
	CustomFields    []CustomField `json:"custom_fields,omitempty"`

	// Internal field for CreateTask only
	Project string `json:"project,omitempty"`
}

// AsanaClient is a client for interacting with the Asana API.
type AsanaClient struct {
	client *resty.Client
}

// NewAsanaClient creates and returns a new AsanaClient using the provided personal access token.
func NewAsanaClient(token string) *AsanaClient {
	c := resty.New()
	c.SetBaseURL("https://app.asana.com/api/1.0")
	c.SetHeader("Authorization", "Bearer "+token)
	c.SetHeader("Accept", "application/json")
	return &AsanaClient{client: c}
}

// CreateTask creates a new task in Asana.
func (ac *AsanaClient) CreateTask(task Task) (*Task, error) {
	payload := map[string]interface{}{
		"data": map[string]interface{}{
			"projects": []string{task.Project},
			"name":     task.Name,
			"notes":    task.Notes,
			"due_on":   task.DueAt,
		},
	}
	if task.StartAt != "" {
		payload["data"].(map[string]interface{})["start_on"] = task.StartAt
	}
	if task.Completed {
		payload["data"].(map[string]interface{})["completed"] = task.Completed
	}

	resp, err := ac.client.R().
		SetBody(payload).
		SetResult(&TaskResponse{}).
		Post("/tasks")
	if err != nil {
		return nil, err
	}
	if resp.IsError() {
		return nil, fmt.Errorf("failed to create task (%d): %s", resp.StatusCode(), resp.String())
	}
	result := resp.Result().(*TaskResponse)
	return &result.Data, nil
}

// UpdateTask updates an existing task in Asana.
func (ac *AsanaClient) UpdateTask(id string, data map[string]interface{}) (*Task, error) {
	payload := map[string]interface{}{"data": data}

	resp, err := ac.client.R().
		SetBody(payload).
		SetResult(&TaskResponse{}).
		SetPathParam("id", id).
		Put("/tasks/{id}")
	if err != nil {
		return nil, err
	}
	if resp.IsError() {
		return nil, fmt.Errorf("failed to update task (%d): %s", resp.StatusCode(), resp.String())
	}
	result := resp.Result().(*TaskResponse)
	return &result.Data, nil
}

// DeleteTask deletes a task in Asana.
func (ac *AsanaClient) DeleteTask(id string) error {
	resp, err := ac.client.R().
		SetPathParam("id", id).
		Delete("/tasks/{id}")
	if err != nil {
		return err
	}
	if resp.IsError() {
		return fmt.Errorf("failed to delete task (%d): %s", resp.StatusCode(), resp.String())
	}
	return nil
}
