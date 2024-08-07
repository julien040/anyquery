package main

type Tasks []Task

type Task struct {
	ID           string      `json:"id,omitempty"`
	AssignerID   *string     `json:"assigner_id,omitempty"`
	AssigneeID   *string     `json:"assignee_id,omitempty"`
	ProjectID    string      `json:"project_id,omitempty"`
	SectionID    *string     `json:"section_id,omitempty"`
	ParentID     interface{} `json:"parent_id,omitempty"`
	Order        int64       `json:"order,omitempty"`
	Content      string      `json:"content,omitempty"`
	Description  string      `json:"description,omitempty"`
	IsCompleted  bool        `json:"is_completed,omitempty"`
	Labels       []string    `json:"labels,omitempty"`
	Priority     int64       `json:"priority,omitempty"`
	CommentCount int64       `json:"comment_count,omitempty"`
	CreatorID    string      `json:"creator_id,omitempty"`
	CreatedAt    string      `json:"created_at,omitempty"`
	Due          *Due        `json:"due,omitempty"`
	URL          string      `json:"url,omitempty"`
	Duration     interface{} `json:"duration,omitempty"`
}

type Due struct {
	Date        string `json:"date"`
	String      string `json:"string"`
	Lang        string `json:"lang"`
	IsRecurring bool   `json:"is_recurring"`
}
