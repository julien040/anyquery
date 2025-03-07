package main

import (
	"fmt"
	"log"

	"github.com/go-resty/resty/v2"
)

type TasksResponse struct {
	Data []Task `json:"data"`
}

type TaskResponse struct {
	Data Task `json:"data"`
}

type TasksQueryNextPage struct {
	Offset string `json:"offset"`
}

type TasksQueryResponse struct {
	Data     []Task             `json:"data"`
	NextPage TasksQueryNextPage `json:"next_page"`
}

// AsanaClient is a client for interacting with the Asana API.
type AsanaClient struct {
	client  *resty.Client
	token   string
	baseURL string
}

// NewAsanaClient creates and returns a new AsanaClient using the provided personal access token.
func NewAsanaClient(token string) *AsanaClient {
	c := resty.New()
	c = c.SetBaseURL("https://app.asana.com/api/1.0")
	c.SetHeader("Authorization", "Bearer "+token)
	c.SetHeader("Accept", "application/json")
	return &AsanaClient{
		client:  c,
		token:   token,
		baseURL: "https://app.asana.com/api/1.0",
	}
}

// Task represents an Asana task with the most common fields.
type Task struct {
	Gid             string        `json:"gid,omitempty"`         // Globally unique task identifier.
	Name            string        `json:"name,omitempty"`        // The name of the task.
	Completed       bool          `json:"completed,omitempty"`   // Task completion status.
	Liked           bool          `json:"liked,omitempty"`       // Task like status.
	Assignee        *User         `json:"assignee,omitempty"`    // The user assigned to the task.
	DueOn           string        `json:"due_on,omitempty"`      // Task due date (YYYY-MM-DD format).
	DueAt           string        `json:"due_at,omitempty"`      // Task due date (YYYY-MM-DDTHH:MM:SS format).
	StartAt         string        `json:"start_at,omitempty"`    // Task start date (YYYY-MM-DD format).
	StartOn         string        `json:"start_on,omitempty"`    // Task start date (YYYY-MM-DDTHH:MM:SS format).
	Parent          string        `json:"parent,omitempty"`      // Parent task identifier.
	Notes           string        `json:"notes,omitempty"`       // Task description or notes.
	Memberships     []Memberships `json:"memberships,omitempty"` // Project memberships.s
	CreatedAt       string        `json:"created_at,omitempty"`  // Timestamp when the task was created.
	UpdatedAt       string        `json:"modified_at,omitempty"` // Timestamp when the task was last modified.
	ResourceSubtype string        `json:"resource_subtype,omitempty"`
	CustomFields    []CustomField `json:"custom_fields,omitempty"`

	// Internal fields for CreateTask and UpdateTask
	Project string `json:"project,omitempty"`
}

type Memberships struct {
	Project ProjectSection `json:"project,omitempty"`
	Section ProjectSection `json:"section,omitempty"`
}

type ProjectSection struct {
	Gid  string `json:"gid,omitempty"`  // Globally unique project identifier.
	Name string `json:"name,omitempty"` // The name of the project.
}

// User represents an Asana user.
type User struct {
	Gid  string `json:"gid"`  // Globally unique user identifier.
	Name string `json:"name"` // The name of the user.
}

type CustomField struct {
	Gid          string  `json:"gid,omitempty"`
	Name         string  `json:"name,omitempty"`
	DisplayValue *string `json:"display_value,omitempty"`
}

// GetTasks retrieves tasks from Asana.
func (ac *AsanaClient) GetTasks() ([]Task, error) {
	resp, err := ac.client.R().
		SetResult(&TasksResponse{}).
		Get("/tasks")
	if err != nil {
		return nil, err
	}
	result := resp.Result().(*TasksResponse)
	return result.Data, nil
}

// CreateTask creates a new task in Asana.
func (ac *AsanaClient) CreateTask(task Task) (*Task, error) {
	payload := map[string]interface{}{
		"data": map[string]interface{}{
			"projects": task.Project,
			"name":     task.Name,
			"notes":    task.Notes,
			"due_on":   task.DueAt,
			// Additional fields can be added here.
			"resource_type": "task",
		},
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
func (ac *AsanaClient) UpdateTask(id string, task Task) (*Task, error) {
	payload := map[string]interface{}{
		"data": map[string]interface{}{
			/* "name":      task.Name,
			"notes":     task.Notes,
			"due_on":    task.DueOn,
			"completed": task.Completed, */
		},
	}
	if task.Name != "" {
		payload["data"].(map[string]interface{})["name"] = task.Name
	}
	if task.Notes != "" {
		payload["data"].(map[string]interface{})["notes"] = task.Notes
	}
	if task.DueAt != "" {
		payload["data"].(map[string]interface{})["due_on"] = task.DueAt
	}
	if task.Completed {
		payload["data"].(map[string]interface{})["completed"] = task.Completed
	}
	if task.Liked {
		payload["data"].(map[string]interface{})["liked"] = task.Liked
	}
	if task.StartAt != "" {
		payload["data"].(map[string]interface{})["start_on"] = task.StartAt
	}

	log.Printf("Payload: %v (%s)", payload, id)
	resp, err := ac.client.R().
		SetBody(payload).
		SetResult(&TaskResponse{}).
		Put(fmt.Sprintf("/tasks/%s", id))
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
		Delete(fmt.Sprintf("/tasks/%s", id))

	if err != nil {
		return err
	}

	if resp.IsError() {
		return fmt.Errorf("failed to delete task (%d): %s", resp.StatusCode(), resp.String())
	}

	return nil
}
