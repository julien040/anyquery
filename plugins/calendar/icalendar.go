package main

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"time"

	ics "github.com/arran4/golang-ical"
	"github.com/julien040/anyquery/rpc"
	"github.com/julien040/go-ternary"
)

// A constructor to create a new table instance
// This function is called everytime a new connection is made to the plugin
//
// It should return a new table instance, the database schema and if there is an error
func icalendarCreator(args rpc.TableCreatorArgs) (rpc.Table, *rpc.DatabaseSchema, error) {
	return &icalendarTable{}, &rpc.DatabaseSchema{
		Columns: []rpc.DatabaseSchemaColumn{
			{
				Name:        "path",
				Type:        rpc.ColumnTypeString,
				IsParameter: true,
				IsRequired:  true,
			},
			{
				Name: "id",
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
				Name: "summary",
				Type: rpc.ColumnTypeString,
			},
			{
				Name: "description",
				Type: rpc.ColumnTypeString,
			},
			{
				Name: "attendees",
				Type: rpc.ColumnTypeString,
			},
			{
				Name: "status",
				Type: rpc.ColumnTypeString,
			},
			{
				Name: "priority",
				Type: rpc.ColumnTypeString,
			},
			{
				Name: "location",
				Type: rpc.ColumnTypeString,
			},
			{
				Name: "geo",
				Type: rpc.ColumnTypeString,
			},
			{
				Name: "organizer",
				Type: rpc.ColumnTypeString,
			},
			{
				Name: "sequence",
				Type: rpc.ColumnTypeInt,
			},
			{
				Name: "created_at",
				Type: rpc.ColumnTypeString,
			},
			{
				Name: "last_modified_at",
				Type: rpc.ColumnTypeString,
			},
		},
	}, nil
}

type icalendarTable struct {
}

type icalendarCursor struct {
}

// Return a slice of rows that will be returned to Anyquery and filtered.
// The second return value is true if the cursor has no more rows to return
//
// The constraints are used for optimization purposes to "pre-filter" the rows
// If the rows returned don't match the constraints, it's not an issue. Anyquery will filter them out
func (t *icalendarCursor) Query(constraints rpc.QueryConstraint) ([][]interface{}, bool, error) {
	// Get the file path from the constraints
	filePath := ""
	for _, constraint := range constraints.Columns {
		if constraint.ColumnID == 0 {
			rawStr, ok := constraint.Value.(string)
			if !ok {
				return nil, true, fmt.Errorf("file path is not a string")
			}
			filePath = rawStr
			break
		}
	}

	if filePath == "" {
		return nil, true, fmt.Errorf("file path is empty. Pass a filepath or a URL as a table parameter")
	}

	isURL := false
	// Check if the path is a URL
	parsed, err := url.Parse(filePath)
	if err == nil && (parsed.Scheme == "http" || parsed.Scheme == "https") && parsed.Host != "" {
		isURL = true
	}

	var ioReader io.Reader

	if isURL {
		// Download the file
		req, err := http.NewRequest("GET", filePath, nil)
		if err != nil {
			return nil, true, fmt.Errorf("failed to create request: %s", err)
		}
		res, err := http.DefaultClient.Do(req)
		if err != nil {
			return nil, true, fmt.Errorf("failed to download file: %s", err)
		}

		if res.StatusCode != http.StatusOK {
			return nil, true, fmt.Errorf("failed to download file (status code: %d)", res.StatusCode)
		}

		ioReader = res.Body
		defer res.Body.Close()
	} else {
		// Read the file
		file, err := os.OpenFile(filePath, os.O_RDONLY, 0644)
		if err != nil {
			return nil, true, fmt.Errorf("failed to open file: %s", err)
		}
		defer file.Close()
		ioReader = file
	}

	doc, err := ics.ParseCalendar(ioReader)
	if err != nil {
		return nil, true, fmt.Errorf("failed to parse calendar: %s", err)
	}

	events := doc.Events()
	rows := make([][]interface{}, 0, len(events))

	for _, event := range events {
		startAt, err := event.GetStartAt()
		startAtVal := ternary.If[interface{}](err == nil, startAt.Format(time.RFC3339), nil)
		endAt, err := event.GetEndAt()
		endAtVal := ternary.If[interface{}](err == nil, endAt.Format(time.RFC3339), nil)

		// Try for each property to get the value
		// Otherwise, set it to nil that will be converted to NULL in the database
		summary := interface{}(nil)
		ianaProp := event.GetProperty(ics.ComponentPropertySummary)
		if ianaProp != nil {
			summary = ianaProp.Value
		}
		description := interface{}(nil)
		ianaProp = event.GetProperty(ics.ComponentPropertyDescription)
		if ianaProp != nil {
			description = ianaProp.Value
		}

		attendees := interface{}(nil)
		ianaProp = event.GetProperty(ics.ComponentPropertyAttendee)
		if ianaProp != nil {
			attendees = ianaProp.Value
		}

		status := interface{}(nil)
		ianaProp = event.GetProperty(ics.ComponentPropertyStatus)
		if ianaProp != nil {
			status = ianaProp.Value
		}

		priority := interface{}(nil)
		ianaProp = event.GetProperty(ics.ComponentPropertyPriority)
		if ianaProp != nil {
			priority = ianaProp.Value
		}

		location := interface{}(nil)
		ianaProp = event.GetProperty(ics.ComponentPropertyLocation)
		if ianaProp != nil {
			location = ianaProp.Value
		}

		geo := interface{}(nil)
		ianaProp = event.GetProperty(ics.ComponentPropertyGeo)
		if ianaProp != nil {
			geo = ianaProp.Value
		}

		created := interface{}(nil)
		ianaProp = event.GetProperty(ics.ComponentPropertyCreated)
		if ianaProp != nil {
			// Parse the date time. Ex: 20190411T090536Z
			createdParsed, err := time.Parse("20060102T150405Z", ianaProp.Value)
			if err == nil {
				created = createdParsed.Format(time.RFC3339)
			}

		}

		lastModifiedAt := interface{}(nil)
		ianaProp = event.GetProperty(ics.ComponentPropertyLastModified)
		if ianaProp != nil {
			// Parse the date time. Ex: 20190411T090536Z
			lastModifiedAtParsed, err := time.Parse("20060102T150405Z", ianaProp.Value)
			if err == nil {
				lastModifiedAt = lastModifiedAtParsed.Format(time.RFC3339)
			}
		}

		organizer := interface{}(nil)
		ianaProp = event.GetProperty(ics.ComponentPropertyOrganizer)
		if ianaProp != nil {
			organizer = ianaProp.Value
		}

		sequence := interface{}(nil)
		ianaProp = event.GetProperty(ics.ComponentPropertySequence)
		if ianaProp != nil {
			// Convert the sequence to an integer
			val, err := strconv.Atoi(ianaProp.Value)
			if err == nil {
				sequence = val
			}
		}

		rows = append(rows, []interface{}{
			event.Id(),
			startAtVal,
			endAtVal,
			summary,
			description,
			attendees,
			status,
			priority,
			location,
			geo,
			organizer,
			sequence,
			created,
			lastModifiedAt,
		})
	}

	return rows, true, nil
}

// Create a new cursor that will be used to read rows
func (t *icalendarTable) CreateReader() rpc.ReaderInterface {
	return &icalendarCursor{}
}

// A slice of rows to insert
func (t *icalendarTable) Insert(rows [][]interface{}) error {
	return nil
}

// A slice of rows to update
// The first element of each row is the primary key
// while the rest are the values to update
// The primary key is therefore present twice
func (t *icalendarTable) Update(rows [][]interface{}) error {
	return nil
}

// A slice of primary keys to delete
func (t *icalendarTable) Delete(primaryKeys []interface{}) error {
	return nil
}

// A destructor to clean up resources
func (t *icalendarTable) Close() error {
	return nil
}
