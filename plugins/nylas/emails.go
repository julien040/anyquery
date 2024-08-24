/*
Copyright 2024 Julien CAGNIART

Permission is hereby granted, free of charge, to any person obtaining a copy of this software and associated documentation files (the “Software”), to deal in the Software without restriction, including without limitation the rights to use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of the Software, and to permit persons to whom the Software is furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED “AS IS”, WITHOUT WARRANTY OF ANY KIND, EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.
*/
package main

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/julien040/anyquery/rpc"
)

// A constructor to create a new table instance
// This function is called everytime a new connection is made to the plugin
//
// It should return a new table instance, the database schema and if there is an error
func emailsCreator(args rpc.TableCreatorArgs) (rpc.Table, *rpc.DatabaseSchema, error) {
	// Get crendentials from the user config
	apiKey, grantID, serverHost, err := getCredentials(args)
	if err != nil {
		return nil, nil, err
	}
	return &emailsTable{
			serverHost: serverHost,
			grantID:    grantID,
			apiKey:     apiKey,
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
					Name: "subject",
					Type: rpc.ColumnTypeString,
				},
				{
					Name: "from",
					Type: rpc.ColumnTypeString,
				},
				{
					Name: "to",
					Type: rpc.ColumnTypeString,
				},
				{
					Name: "cc",
					Type: rpc.ColumnTypeString,
				},
				{
					Name: "bcc",
					Type: rpc.ColumnTypeString,
				},
				{
					Name: "reply_to",
					Type: rpc.ColumnTypeString,
				},
				{
					Name: "sent_at",
					Type: rpc.ColumnTypeString,
				},
				{
					Name: "folders",
					Type: rpc.ColumnTypeString,
				},
				{
					Name: "starred",
					Type: rpc.ColumnTypeBool,
				},
				{
					Name: "unread",
					Type: rpc.ColumnTypeBool,
				},
				{
					Name: "body",
					Type: rpc.ColumnTypeString,
				},
			},
		}, nil
}

type emailsTable struct {
	serverHost string
	grantID    string
	apiKey     string
}

type emailsCursor struct {
	serverHost string
	grantID    string
	apiKey     string
	nextCursor string
}

// Return a slice of rows that will be returned to Anyquery and filtered.
// The second return value is true if the cursor has no more rows to return
//
// The constraints are used for optimization purposes to "pre-filter" the rows
// If the rows returned don't match the constraints, it's not an issue. Anyquery will filter them out
func (t *emailsCursor) Query(constraints rpc.QueryConstraint) ([][]interface{}, bool, error) {

	// Extract query constraints
	var subject string
	var unread, starred interface{}

	for _, c := range constraints.Columns {
		switch c.ColumnID {
		case 1:
			subject, _ = c.Value.(string)
		case 2:
			raw, ok := c.Value.(int64)
			if ok {
				unread = raw == 1
			}
		case 3:
			raw, ok := c.Value.(int64)
			if ok {
				starred = raw == 1
			}
		}
	}

	// Get the next page of emails
	endpoint := fmt.Sprintf("https://%s/v3/grants/{grant_id}/messages", t.serverHost)
	data := &EmailsResponse{}
	req := client.R().
		SetHeader("Authorization", "Bearer "+t.apiKey).
		SetPathParam("grant_id", t.grantID).
		SetQueryParam("limit", "20").
		SetResult(data)

	if unread != nil {
		req.SetQueryParam("unread", fmt.Sprintf("%v", unread))
	}

	if starred != nil {
		req.SetQueryParam("starred", fmt.Sprintf("%v", starred))
	}

	if subject != "" {
		req.SetQueryParam("subject", subject)
	}

	if t.nextCursor != "" {
		req.SetQueryParam("page_token", t.nextCursor)
	}

	resp, err := req.Get(endpoint)
	if err != nil {
		return nil, true, fmt.Errorf("error fetching emails: %v", err)
	}

	if resp.IsError() {
		return nil, true, fmt.Errorf("error fetching emails (code: %d): %s", resp.StatusCode(), resp.String())
	}

	// Update the cursor
	t.nextCursor = data.NextCursor

	// Prepare the rows
	rows := make([][]interface{}, 0, len(data.Data))
	for _, email := range data.Data {
		row := []interface{}{
			email.ID,
			email.Subject,
			convertFromToArray(email.From),
			convertFromToArray(email.To),
			convertFromToArray(email.Cc),
			convertFromToArray(email.Bcc),
			convertFromToArray(email.ReplyTo),
			time.Unix(email.Date, 0).Format(time.RFC3339),
			email.Folders,
			email.Starred,
			email.Unread,
			email.Body,
		}
		rows = append(rows, row)
	}

	return rows, true, nil
}

// Create a new cursor that will be used to read rows
func (t *emailsTable) CreateReader() rpc.ReaderInterface {
	return &emailsCursor{
		serverHost: t.serverHost,
		grantID:    t.grantID,
		apiKey:     t.apiKey,
		nextCursor: "",
	}
}

// A slice of rows to insert
func (t *emailsTable) Insert(rows [][]interface{}) error {
	for _, row := range rows {
		reqBody := &Message{}
		// Extract the subject, to, body
		subject := getString(row, 1)
		if subject == "" {
			subject = "No subject"
		}
		reqBody.Subject = subject

		// Extract the to
		to := getString(row, 3)
		if to != "" {
			// Try to parse it as an array of strings
			val := []string{}
			if err := json.Unmarshal([]byte(to), &val); err == nil {
				for _, v := range val {
					reqBody.To = append(reqBody.To, From{Email: v})
				}
			} else {
				reqBody.To = []From{{Email: to}}
			}
		} else {
			return fmt.Errorf("to is required")
		}

		// Extract the body
		reqBody.Body = getString(row, 11)
		reqBody.Body = reqBody.Body + "<br><br> Sent from <a href='https://anyquery.dev'>Anyquery</a>"

		// Send the email
		endpoint := fmt.Sprintf("https://%s/v3/grants/{grant_id}/messages/send", t.serverHost)
		req := client.R().
			SetPathParam("grant_id", t.grantID).
			SetHeader("Authorization", "Bearer "+t.apiKey).
			SetBody(reqBody)

		resp, err := req.Post(endpoint)
		if err != nil {
			return fmt.Errorf("error while sending email: %s", err)
		}

		if resp.IsError() {
			return fmt.Errorf("error while sending email(code: %d): %s", resp.StatusCode(), resp.String())
		}

	}

	return nil
}

// A slice of rows to update
// The first element of each row is the primary key
// while the rest are the values to update
// The primary key is therefore present twice
func (t *emailsTable) Update(rows [][]interface{}) error {
	return nil
}

// A slice of primary keys to delete
func (t *emailsTable) Delete(primaryKeys []interface{}) error {
	for _, pk := range primaryKeys {
		if _, ok := pk.(string); !ok {
			return fmt.Errorf("primary key is not a string")
		}

		endpoint := fmt.Sprintf("https://%s/v3/grants/{grant_id}/messages/{message_id}", t.serverHost)
		resp, err := client.R().
			SetPathParam("grant_id", t.grantID).
			SetPathParam("message_id", pk.(string)).
			SetHeader("Authorization", "Bearer "+t.apiKey).
			Delete(endpoint)

		if err != nil {
			return fmt.Errorf("error while deleting email: %s", err)
		}

		if resp.IsError() {
			return fmt.Errorf("error while deleting email(code: %d): %s", resp.StatusCode(), resp.String())
		}
	}
	return nil
}

// A destructor to clean up resources
func (t *emailsTable) Close() error {
	return nil
}

func convertFromToArray(data []From) interface{} {
	if len(data) == 0 {
		return nil
	}
	var result []string
	for _, v := range data {
		result = append(result, v.Email)
	}
	return result
}
