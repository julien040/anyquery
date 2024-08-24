/*
Copyright 2024 Julien CAGNIART

Permission is hereby granted, free of charge, to any person obtaining a copy of this software and associated documentation files (the “Software”), to deal in the Software without restriction, including without limitation the rights to use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of the Software, and to permit persons to whom the Software is furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED “AS IS”, WITHOUT WARRANTY OF ANY KIND, EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.
*/
package main

type EventsResponse struct {
	RequestID  string  `json:"request_id,omitempty"`
	Data       []Event `json:"data,omitempty"`
	NextCursor string  `json:"next_cursor,omitempty"`
}

type Event struct {
	Busy             bool              `json:"busy,omitempty"`
	CalendarID       string            `json:"calendar_id,omitempty"`
	Conferencing     *Conferencing     `json:"conferencing,omitempty"`
	CreatedAt        int64             `json:"created_at,omitempty"`
	Description      string            `json:"description,omitempty"`
	HideParticipants bool              `json:"hide_participants,omitempty"`
	GrantID          string            `json:"grant_id,omitempty"`
	HTMLLink         string            `json:"html_link,omitempty"`
	ID               string            `json:"id,omitempty"`
	Location         string            `json:"location,omitempty"`
	Metadata         map[string]string `json:"metadata,omitempty"`
	Object           string            `json:"object,omitempty"`
	Organizer        *Organizer        `json:"organizer,omitempty"`
	Participants     []Participant     `json:"participants,omitempty"`
	ReadOnly         bool              `json:"read_only,omitempty"`
	Reminders        *Reminders        `json:"reminders,omitempty"`
	Status           string            `json:"status,omitempty"`
	Title            string            `json:"title,omitempty"`
	UpdatedAt        int64             `json:"updated_at,omitempty"`
	Visibility       string            `json:"visibility,omitempty"`
	When             When              `json:"when,omitempty"`
}

type Conferencing struct {
	Provider string  `json:"provider,omitempty"`
	Details  Details `json:"details,omitempty"`
}

type Details struct {
	MeetingCode string `json:"meeting_code,omitempty"`
	Password    string `json:"password,omitempty"`
	URL         string `json:"url,omitempty"`
}

type Organizer struct {
	Email string `json:"email,omitempty"`
	Name  string `json:"name,omitempty"`
}

type Participant struct {
	Comment     string `json:"comment,omitempty"`
	Email       string `json:"email,omitempty"`
	Name        string `json:"name,omitempty"`
	PhoneNumber string `json:"phone_number,omitempty"`
	Status      string `json:"status,omitempty"`
}

type Reminders struct {
	UseDefault bool       `json:"use_default,omitempty"`
	Overrides  []Override `json:"overrides,omitempty"`
}

type Override struct {
	ReminderMinutes int64  `json:"reminder_minutes,omitempty"`
	ReminderMethod  string `json:"reminder_method,omitempty"`
}

type When struct {
	StartTime     int64  `json:"start_time,omitempty"`
	EndTime       int64  `json:"end_time,omitempty"`
	StartTimezone string `json:"start_timezone,omitempty"`
	EndTimezone   string `json:"end_timezone,omitempty"`
	Date          string `json:"date,omitempty"`
	StartDate     string `json:"start_date,omitempty"`
	EndDate       string `json:"end_date,omitempty"`
}

type EmailsResponse struct {
	RequestID  string    `json:"request_id,omitempty"`
	Data       []Message `json:"data,omitempty"`
	NextCursor string    `json:"next_cursor,omitempty"`
}

type Message struct {
	Starred     bool         `json:"starred,omitempty"`
	Unread      bool         `json:"unread,omitempty"`
	Folders     []string     `json:"folders,omitempty"`
	Subject     string       `json:"subject,omitempty"`
	ThreadID    string       `json:"thread_id,omitempty"`
	Body        string       `json:"body,omitempty"`
	GrantID     string       `json:"grant_id,omitempty"`
	ID          string       `json:"id,omitempty"`
	Object      string       `json:"object,omitempty"`
	Snippet     string       `json:"snippet,omitempty"`
	Bcc         []From       `json:"bcc,omitempty"`
	Cc          []From       `json:"cc,omitempty"`
	From        []From       `json:"from,omitempty"`
	ReplyTo     []From       `json:"reply_to,omitempty"`
	To          []From       `json:"to,omitempty"`
	Date        int64        `json:"date,omitempty"`
	Attachments []Attachment `json:"attachments,omitempty"`
}

type Attachment struct {
	IsInline    bool   `json:"is_inline,omitempty"`
	ID          string `json:"id,omitempty"`
	GrantID     string `json:"grant_id,omitempty"`
	Filename    string `json:"filename,omitempty"`
	ContentType string `json:"content_type,omitempty"`
	ContentID   string `json:"content_id,omitempty"`
	Size        int64  `json:"size,omitempty"`
}

type From struct {
	Name  string `json:"name,omitempty"`
	Email string `json:"email,omitempty"`
}
