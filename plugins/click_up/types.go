package main

type Tasks struct {
	Tasks    []Task `json:"tasks"`
	LastPage bool   `json:"last_page"`
}

type Task struct {
	ID                  string        `json:"id"`
	CustomID            interface{}   `json:"custom_id"`
	CustomItemID        int64         `json:"custom_item_id"`
	Name                string        `json:"name"`
	TextContent         *string       `json:"text_content"`
	Description         *string       `json:"description"`
	MarkdownDescription string        `json:"markdown_description"`
	Status              Status        `json:"status"`
	Orderindex          string        `json:"orderindex"`
	DateCreated         string        `json:"date_created"`
	DateUpdated         string        `json:"date_updated"`
	DateClosed          *string       `json:"date_closed"`
	DateDone            *string       `json:"date_done"`
	Archived            bool          `json:"archived"`
	Creator             Person        `json:"creator"`
	Assignees           []Person      `json:"assignees"`
	GroupAssignees      []interface{} `json:"group_assignees"`
	Watchers            []Person      `json:"watchers"`
	Checklists          []Checklist   `json:"checklists"`
	Tags                []Tag         `json:"tags"`
	Parent              *string       `json:"parent"`
	Priority            *Priority     `json:"priority"`
	DueDate             *string       `json:"due_date"`
	StartDate           *string       `json:"start_date"`
	Points              interface{}   `json:"points"`
	TimeEstimate        interface{}   `json:"time_estimate"`
	CustomFields        []CustomField `json:"custom_fields"`
	Dependencies        []interface{} `json:"dependencies"`
	LinkedTasks         []LinkedTask  `json:"linked_tasks"`
	Locations           []interface{} `json:"locations"`
	TeamID              string        `json:"team_id"`
	URL                 string        `json:"url"`
	Sharing             Sharing       `json:"sharing"`
	PermissionLevel     string        `json:"permission_level"`
	List                Folder        `json:"list"`
	Project             Folder        `json:"project"`
	Folder              Folder        `json:"folder"`
	Space               Space         `json:"space"`
	TimeSpent           *int64        `json:"time_spent,omitempty"`
}

type Person struct {
	ID             int64   `json:"id"`
	Username       string  `json:"username"`
	Color          string  `json:"color"`
	Initials       *string `json:"initials,omitempty"`
	Email          string  `json:"email"`
	ProfilePicture string  `json:"profilePicture"`
}

type Checklist struct {
	ID          string `json:"id"`
	TaskID      string `json:"task_id"`
	Name        string `json:"name"`
	DateCreated string `json:"date_created"`
	Orderindex  int64  `json:"orderindex"`
	Creator     int64  `json:"creator"`
	Resolved    int64  `json:"resolved"`
	Unresolved  int64  `json:"unresolved"`
	Items       []Item `json:"items"`
}

type Item struct {
	ID            string        `json:"id"`
	Name          string        `json:"name"`
	Orderindex    int64         `json:"orderindex"`
	Assignee      interface{}   `json:"assignee"`
	GroupAssignee interface{}   `json:"group_assignee"`
	Resolved      bool          `json:"resolved"`
	Parent        interface{}   `json:"parent"`
	DateCreated   string        `json:"date_created"`
	Children      []interface{} `json:"children"`
}

type CustomField struct {
	ID             string      `json:"id"`
	Name           string      `json:"name"`
	Type           string      `json:"type"`
	TypeConfig     TypeConfig  `json:"type_config"`
	DateCreated    string      `json:"date_created"`
	HideFromGuests bool        `json:"hide_from_guests"`
	Required       bool        `json:"required"`
	Value          interface{} `json:"value,omitempty"`
	ValueRichtext  interface{} `json:"value_richtext"`
}

type TypeConfig struct {
	Precision    int64  `json:"precision"`
	CurrencyType string `json:"currency_type"`
}

type Folder struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Hidden *bool  `json:"hidden,omitempty"`
	Access bool   `json:"access"`
}

type LinkedTask struct {
	TaskID      string `json:"task_id"`
	LinkID      string `json:"link_id"`
	DateCreated string `json:"date_created"`
	Userid      string `json:"userid"`
	WorkspaceID string `json:"workspace_id"`
}

type Priority struct {
	Color      string `json:"color"`
	ID         string `json:"id"`
	Orderindex string `json:"orderindex"`
	Priority   string `json:"priority"`
}

type Sharing struct {
	Public               bool        `json:"public"`
	PublicShareExpiresOn interface{} `json:"public_share_expires_on"`
	PublicFields         []string    `json:"public_fields"`
	Token                interface{} `json:"token"`
	SEOOptimized         bool        `json:"seo_optimized"`
}

type Space struct {
	ID string `json:"id"`
}

type Status struct {
	Status     string `json:"status"`
	ID         string `json:"id"`
	Color      string `json:"color"`
	Type       string `json:"type"`
	Orderindex int64  `json:"orderindex"`
}

type Tag struct {
	Name    string `json:"name"`
	TagFg   string `json:"tag_fg"`
	TagBg   string `json:"tag_bg"`
	Creator int64  `json:"creator"`
}

// Docs
type Docs struct {
	Docs       []Doc  `json:"docs"`
	NextCursor string `json:"next_cursor"`
}

type Doc struct {
	ID          string    `json:"id"`
	DateCreated int64     `json:"date_created"`
	DateUpdated int64     `json:"date_updated"`
	Name        string    `json:"name"`
	Parent      ParentDoc `json:"parent"`
	WorkspaceID int64     `json:"workspace_id"`
	Creator     int64     `json:"creator"`
	Deleted     bool      `json:"deleted"`
	Type        int64     `json:"type"`
}

type ParentDoc struct {
	ID   string `json:"id"`
	Type int64  `json:"type"`
}

type Pages []Page

type Page struct {
	ID                  string              `json:"id"`
	DocID               string              `json:"doc_id"`
	WorkspaceID         int64               `json:"workspace_id"`
	Name                string              `json:"name"`
	DateCreated         int64               `json:"date_created"`
	DateUpdated         int64               `json:"date_updated"`
	Content             string              `json:"content"`
	CreatorID           int64               `json:"creator_id"`
	Deleted             bool                `json:"deleted"`
	DateEdited          int64               `json:"date_edited"`
	EditedBy            int64               `json:"edited_by"`
	Archived            bool                `json:"archived"`
	Protected           bool                `json:"protected"`
	PresentationDetails PresentationDetails `json:"presentation_details"`
}

type PresentationDetails struct {
	ShowContributorHeader bool `json:"show_contributor_header"`
}

type Folders struct {
	Folders []FolderItem `json:"folders"`
}

type FolderItem struct {
	ID               string        `json:"id"`
	Name             string        `json:"name"`
	Orderindex       int64         `json:"orderindex"`
	OverrideStatuses bool          `json:"override_statuses"`
	Hidden           bool          `json:"hidden"`
	Space            FolderSpace   `json:"space"`
	TaskCount        string        `json:"task_count"`
	Archived         bool          `json:"archived"`
	Statuses         []interface{} `json:"statuses"`
	Lists            []List        `json:"lists"`
	PermissionLevel  string        `json:"permission_level"`
}

type ListSpace struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Access bool   `json:"access"`
}

type StatusList struct {
	ID          string `json:"id"`
	Status      string `json:"status"`
	Orderindex  int64  `json:"orderindex"`
	Color       string `json:"color"`
	Type        string `json:"type"`
	StatusGroup string `json:"status_group"`
}

type FolderSpace struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// Lists
type Lists struct {
	Lists []List `json:"lists"`
}

type List struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Orderindex int64  `json:"orderindex"`
	Content    string `json:"content"`
	/* Status           StatusList2  `json:"status"`
	Priority         PriorityList `json:"priority"` */
	Assignee string `json:"assignee"`
	// TaskCount is an interface because the docs says it's a string, but it returns an int
	TaskCount        interface{} `json:"task_count"`
	DueDate          *string     `json:"due_date"`
	StartDate        *string     `json:"start_date"`
	Folder           FolderList  `json:"folder"`
	Space            FolderList  `json:"space"`
	Archived         bool        `json:"archived"`
	OverrideStatuses bool        `json:"override_statuses"`
	PermissionLevel  string      `json:"permission_level"`
}

type FolderList struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Hidden *bool  `json:"hidden,omitempty"`
	Access bool   `json:"access"`
}

type PriorityList struct {
	Priority string `json:"priority"`
	Color    string `json:"color"`
}

type StatusList2 struct {
	Status    string `json:"status"`
	Color     string `json:"color"`
	HideLabel bool   `json:"hide_label"`
}

// Whoami

type User struct {
	User UserClass `json:"user"`
}

type UserClass struct {
	ID                int64  `json:"id"`
	Username          string `json:"username"`
	Email             string `json:"email"`
	Color             string `json:"color"`
	ProfilePicture    string `json:"profilePicture"`
	Initials          string `json:"initials"`
	WeekStartDay      int64  `json:"week_start_day"`
	GlobalFontSupport bool   `json:"global_font_support"`
	Timezone          string `json:"timezone"`
}
