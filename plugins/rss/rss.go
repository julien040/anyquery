package main

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"time"

	"github.com/julien040/anyquery/rpc"
	"github.com/mmcdole/gofeed"
)

// A constructor to create a new table instance
// This function is called everytime a new connection is made to the plugin
//
// It should return a new table instance, the database schema and if there is an error
func rssCreator(args rpc.TableCreatorArgs) (rpc.Table, *rpc.DatabaseSchema, error) {
	return &rssTable{}, &rpc.DatabaseSchema{
		PrimaryKey: 1,
		Columns: []rpc.DatabaseSchemaColumn{
			{
				Name:        "path",
				Type:        rpc.ColumnTypeString,
				IsParameter: true,
				IsRequired:  true,
			},
			{
				Name: "guid",
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
				Name: "content",
				Type: rpc.ColumnTypeString,
			},
			{
				Name: "links",
				Type: rpc.ColumnTypeString,
			},
			{
				Name: "updated",
				Type: rpc.ColumnTypeString,
			},
			{
				Name: "published",
				Type: rpc.ColumnTypeString,
			},
			{
				Name: "authors",
				Type: rpc.ColumnTypeString,
			},
			{
				Name: "image_url",
				Type: rpc.ColumnTypeString,
			},
			{
				Name: "image_title",
				Type: rpc.ColumnTypeString,
			},
			{
				Name: "categories",
				Type: rpc.ColumnTypeString,
			},
			{
				Name: "enclosures",
				Type: rpc.ColumnTypeString,
			},
		},
	}, nil
}

type rssTable struct {
}

type rssCursor struct {
}

// Return a slice of rows that will be returned to Anyquery and filtered.
// The second return value is true if the cursor has no more rows to return
//
// The constraints are used for optimization purposes to "pre-filter" the rows
// If the rows returned don't match the constraints, it's not an issue. Anyquery will filter them out
func (t *rssCursor) Query(constraints rpc.QueryConstraint) ([][]interface{}, bool, error) {
	// Get the path or the URL from the constraints
	path := ""
	for _, c := range constraints.Columns {
		if c.ColumnID == 0 {
			rawStr, ok := c.Value.(string)
			if !ok {
				return nil, true, fmt.Errorf("path is not a string")
			}
			path = rawStr
		}
	}
	if path == "" {
		return nil, true, fmt.Errorf("path is empty")
	}

	isURL := false

	parsed, err := url.ParseRequestURI(path)
	if err == nil && parsed.Scheme != "" {
		isURL = true
	}

	var feed *gofeed.Feed
	parser := gofeed.NewParser()
	if isURL {
		feed, err = parser.ParseURL(path)
		if err != nil {
			return nil, true, fmt.Errorf("error fetching/reading feed: %v", err)
		}

	} else {
		file, err := os.Open(path)
		if err != nil {
			return nil, true, fmt.Errorf("error opening file: %v", err)
		}
		defer file.Close()
		feed, err = parser.Parse(file)
		if err != nil {
			return nil, true, fmt.Errorf("error reading feed: %v", err)
		}
	}

	rows := make([][]interface{}, 0, len(feed.Items))
	for _, item := range feed.Items {
		row := make([]interface{}, 0, 13)
		row = append(row, item.GUID)
		row = append(row, item.Title)
		row = append(row, item.Description)
		row = append(row, item.Content)
		row = append(row, item.Links)
		if item.UpdatedParsed != nil {
			row = append(row, item.UpdatedParsed.Format(time.RFC3339))
		} else {
			row = append(row, nil)
		}
		if item.PublishedParsed != nil {
			row = append(row, item.PublishedParsed.Format(time.RFC3339))
		} else {
			row = append(row, nil)
		}
		// Serialize authors as json
		serializedAuthors, err := json.Marshal(item.Authors)
		if err != nil || len(item.Authors) == 0 {
			row = append(row, nil)
		} else {
			row = append(row, string(serializedAuthors))
		}
		if item.Image != nil {
			row = append(row, item.Image.URL)
			row = append(row, item.Image.Title)
		} else {
			row = append(row, nil)
			row = append(row, nil)
		}
		if item.Categories == nil {
			row = append(row, nil)
		} else {
			row = append(row, item.Categories)
		}
		// Serialize enclosures as json
		serializedEnclosures, err := json.Marshal(item.Enclosures)
		if err != nil || len(item.Enclosures) == 0 {
			row = append(row, nil)
		} else {
			row = append(row, string(serializedEnclosures))
		}
		rows = append(rows, row)
	}

	return rows, true, nil

}

// Create a new cursor that will be used to read rows
func (t *rssTable) CreateReader() rpc.ReaderInterface {
	return &rssCursor{}
}

// A slice of rows to insert
func (t *rssTable) Insert(rows [][]interface{}) error {
	return nil
}

// A slice of rows to update
// The first element of each row is the primary key
// while the rest are the values to update
// The primary key is therefore present twice
func (t *rssTable) Update(rows [][]interface{}) error {
	return nil
}

// A slice of primary keys to delete
func (t *rssTable) Delete(primaryKeys []interface{}) error {
	return nil
}

// A destructor to clean up resources
func (t *rssTable) Close() error {
	return nil
}
