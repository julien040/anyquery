/*
Copyright 2024 Julien CAGNIART

Permission is hereby granted, free of charge, to any person obtaining a copy of this software and associated documentation files (the “Software”), to deal in the Software without restriction, including without limitation the rights to use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of the Software, and to permit persons to whom the Software is furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED “AS IS”, WITHOUT WARRANTY OF ANY KIND, EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.
*/
package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/julien040/anyquery/rpc"
)

// A constructor to create a new table instance
// This function is called everytime a new connection is made to the plugin
//
// It should return a new table instance, the database schema and if there is an error
func eventsCreator(args rpc.TableCreatorArgs) (rpc.Table, *rpc.DatabaseSchema, error) {
	// Get crendentials from the user config
	apiKey, grantID, serverHost, err := getCredentials(args)
	if err != nil {
		return nil, nil, err
	}

	return &eventsTable{
			serverHost: serverHost,
			grantID:    grantID,
			apiKey:     apiKey,
		}, &rpc.DatabaseSchema{
			HandlesInsert: true,
			HandlesUpdate: true,
			HandlesDelete: true,
			PrimaryKey:    1,
			Columns: []rpc.DatabaseSchemaColumn{
				{
					Name:        "calendar_id",
					Type:        rpc.ColumnTypeString,
					IsParameter: true,
				},
				{
					// Contraction of the calendar_id and the event_id
					Name: "event_id",
					Type: rpc.ColumnTypeString,
				},
				{
					Name: "title",
					Type: rpc.ColumnTypeString,
				},
				{
					Name: "description",
					Type: rpc.ColumnTypeString,
				},
				{
					Name: "created_at",
					Type: rpc.ColumnTypeString,
				},
				{
					Name: "start_at",
					Type: rpc.ColumnTypeString,
				},
				{
					Name: "end_at",
					Type: rpc.ColumnTypeString,
				},
				{
					Name: "location",
					Type: rpc.ColumnTypeString,
				},
				{
					Name: "status",
					Type: rpc.ColumnTypeString,
				},
				{
					Name: "busy",
					Type: rpc.ColumnTypeBool,
				},
				{
					Name: "link",
					Type: rpc.ColumnTypeString,
				},
				{
					Name: "organizer_email",
					Type: rpc.ColumnTypeString,
				},
				{
					Name: "organizer_name",
					Type: rpc.ColumnTypeString,
				},
			},
		}, nil
}

type eventsTable struct {
	serverHost string
	grantID    string
	apiKey     string
}

type eventsCursor struct {
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
func (t *eventsCursor) Query(constraints rpc.QueryConstraint) ([][]interface{}, bool, error) {
	// Request the events from the Nylas API
	endpoint := fmt.Sprintf("https://%s/v3/grants/{grant_id}/events", t.serverHost)
	data := &EventsResponse{}

	// We will fetch the events 90 days in the past and one year in the future
	before := fmt.Sprintf("%d", time.Now().AddDate(0, 0, -90).Unix())
	future := fmt.Sprintf("%d", time.Now().AddDate(1, 0, 0).Unix())

	// Extract the calendar_id
	calendarID := ""
	var ok bool
	for _, c := range constraints.Columns {
		if c.ColumnID == 0 {
			if calendarID, ok = c.Value.(string); !ok {
				return nil, true, fmt.Errorf("calendar_id is not a string")
			}
			break
		}
	}
	// Default to the primary calendar
	if calendarID == "" {
		calendarID = "primary"
	}

	req := client.R().
		SetPathParam("grant_id", t.grantID).
		SetHeader("Authorization", "Bearer "+t.apiKey).
		SetQueryParams(map[string]string{
			"start":       before,
			"end":         future,
			"limit":       "200",
			"calendar_id": calendarID,
		}).
		SetResult(data)

	if t.nextCursor != "" {
		req.SetQueryParam("page_token", t.nextCursor)
	}

	// Make the request
	resp, err := req.Get(endpoint)
	if err != nil {
		return nil, true, fmt.Errorf("error while fetching events: %s", err)
	}

	if resp.IsError() {
		return nil, true, fmt.Errorf("error while fetching events(code: %d): %s", resp.StatusCode(), resp.String())
	}

	// Update the cursor
	t.nextCursor = data.NextCursor

	rows := make([][]interface{}, 0, len(data.Data))

	for _, event := range data.Data {
		startDate := interface{}(nil)
		if event.When.StartTime != 0 {
			startDate = time.Unix(event.When.StartTime, 0).Format(time.RFC3339)
		} else if event.When.Date != "" {
			startDate = event.When.Date
		} else if event.When.StartDate != "" {
			startDate = event.When.StartDate
		}

		endDate := interface{}(nil)
		if event.When.EndTime != 0 {
			endDate = time.Unix(event.When.EndTime, 0).Format(time.RFC3339)
		} else if event.When.Date != "" {
			endDate = event.When.Date
		} else if event.When.EndDate != "" {
			endDate = event.When.EndDate
		}
		rows = append(rows, []interface{}{
			fmt.Sprintf("%s::::%s", event.CalendarID, event.ID),
			event.Title,
			event.Description,
			time.Unix(event.CreatedAt, 0).Format(time.RFC3339),
			startDate,
			endDate,
			event.Location,
			event.Status,
			event.Busy,
			event.HTMLLink,
			event.Organizer.Email,
			event.Organizer.Name,
		})
	}

	return rows, t.nextCursor == "" || len(data.Data) < 200, nil
}

// Create a new cursor that will be used to read rows
func (t *eventsTable) CreateReader() rpc.ReaderInterface {
	return &eventsCursor{
		serverHost: t.serverHost,
		grantID:    t.grantID,
		apiKey:     t.apiKey,
		nextCursor: "",
	}
}

// A slice of rows to insert
func (t *eventsTable) Insert(rows [][]interface{}) error {
	for _, row := range rows {
		calendarID := getString(row, 0)
		if calendarID == "" {
			calendarID = "primary"
		}
		reqBody := &Event{
			Title:       getString(row, 2),
			Description: getString(row, 3),
			Location:    getString(row, 6),
			Busy:        getInt(row, 7) == 1,
		}

		// Parse the dates
		start := parseDate(row, 5)
		end := parseDate(row, 6)
		if start != 0 && end != 0 {
			reqBody.When = When{
				StartTime: start,
				EndTime:   end,
			}
		} else if start != 0 {
			reqBody.When = When{
				Date: time.Unix(start, 0).Format(time.DateOnly),
			}
		} else {
			// Default to today
			reqBody.When = When{
				Date: time.Now().Format(time.DateOnly),
			}
		}

		// Create the event
		endpoint := fmt.Sprintf("https://%s/v3/grants/{grant_id}/events", t.serverHost)
		resp, err := client.R().
			SetPathParam("grant_id", t.grantID).
			SetHeader("Authorization", "Bearer "+t.apiKey).
			SetQueryParam("calendar_id", calendarID).
			SetBody(reqBody).
			Post(endpoint)

		if err != nil {
			return fmt.Errorf("error while creating event: %s", err)
		}

		if resp.IsError() {
			return fmt.Errorf("error while creating event(code: %d): %s", resp.StatusCode(), resp.String())
		}

	}

	return nil
}

// A slice of rows to update
// The first element of each row is the primary key
// while the rest are the values to update
// The primary key is therefore present twice
func (t *eventsTable) Update(rows [][]interface{}) error {
	for _, row := range rows {
		id := getString(row, 0)
		// The first element is the primary key
		row := row[1:]
		calendarID := getString(row, 0)
		if calendarID == "" {
			calendarID = "primary"
		}
		reqBody := &Event{
			Title:       getString(row, 2),
			Description: getString(row, 3),
			Location:    getString(row, 6),
			Busy:        getInt(row, 7) == 1,
		}

		// Parse the dates
		start := parseDate(row, 5)
		end := parseDate(row, 6)
		if start != 0 && end != 0 {
			reqBody.When = When{
				StartTime: start,
				EndTime:   end,
			}
		} else if start != 0 {
			reqBody.When = When{
				Date: time.Unix(start, 0).Format(time.DateOnly),
			}
		} else {
			// Default to today
			reqBody.When = When{
				Date: time.Now().Format(time.DateOnly),
			}
		}

		// Get the event_id
		if id == "" {
			return fmt.Errorf("event_id is empty")
		}

		// Split the ids
		parts := strings.Split(id, "::::")
		if len(parts) != 2 {
			return fmt.Errorf("invalid event_id format")
		}
		calendarID = parts[0]

		// Update the event
		endpoint := fmt.Sprintf("https://%s/v3/grants/{grant_id}/events/{event_id}", t.serverHost)
		resp, err := client.R().
			SetPathParam("grant_id", t.grantID).
			SetPathParam("event_id", parts[1]).
			SetHeader("Authorization", "Bearer "+t.apiKey).
			SetQueryParam("calendar_id", calendarID).
			SetBody(reqBody).
			Put(endpoint)

		if err != nil {
			return fmt.Errorf("error while updating event: %s", err)
		}

		if resp.IsError() {
			return fmt.Errorf("error while updating event(code: %d): %s", resp.StatusCode(), resp.String())
		}

	}

	return nil
}

// A slice of primary keys to delete
func (t *eventsTable) Delete(primaryKeys []interface{}) error {
	for _, pk := range primaryKeys {

		var calendarID, eventID string
		if pk == nil {
			return fmt.Errorf("primary key is nil")
		}

		if pkStr, ok := pk.(string); ok {
			parts := strings.Split(pkStr, "::::")
			if len(parts) != 2 {
				return fmt.Errorf("invalid primary key format")
			}
			calendarID = parts[0]
			eventID = parts[1]
		} else {
			return fmt.Errorf("primary key is not a string")
		}

		// Update the event
		endpoint := fmt.Sprintf("https://%s/v3/grants/{grant_id}/events/{event_id}", t.serverHost)
		resp, err := client.R().
			SetPathParam("grant_id", t.grantID).
			SetPathParam("event_id", eventID).
			SetHeader("Authorization", "Bearer "+t.apiKey).
			SetQueryParam("calendar_id", calendarID).
			Delete(endpoint)

		if err != nil {
			return fmt.Errorf("error while updating event: %s", err)
		}

		if resp.IsError() {
			return fmt.Errorf("error while updating event(code: %d): %s", resp.StatusCode(), resp.String())
		}

	}
	return nil
}

// A destructor to clean up resources
func (t *eventsTable) Close() error {
	return nil
}
