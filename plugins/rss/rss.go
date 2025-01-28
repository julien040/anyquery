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
				Description: "The path of the RSS feed. Can be a URL or a local file",
			},
			{
				Name:        "guid",
				Type:        rpc.ColumnTypeString,
				Description: "The GUID of the item in the feed",
			},
			{
				Name:        "title",
				Type:        rpc.ColumnTypeString,
				Description: "The title of the item in the feed",
			},
			{
				Name:        "description",
				Type:        rpc.ColumnTypeString,
				Description: "The description of the item in the feed",
			},
			{
				Name:        "content",
				Type:        rpc.ColumnTypeString,
				Description: "The content of the item in the feed",
			},
			{
				Name:        "links",
				Type:        rpc.ColumnTypeJSON,
				Description: "A JSON array of links in the item",
			},
			{
				Name:        "updated_at",
				Type:        rpc.ColumnTypeDateTime,
				Description: "The last time the item was updated (RFC 3339)",
			},
			{
				Name:        "published_at",
				Type:        rpc.ColumnTypeDateTime,
				Description: "The date the item was published (RFC 3339)",
			},
			{
				Name:        "authors",
				Type:        rpc.ColumnTypeJSON,
				Description: "A JSON array of object authors (fields: name and email)",
			},
			{
				Name:        "image_url",
				Type:        rpc.ColumnTypeString,
				Description: "The URL of the image associated with the item, if any",
			},
			{
				Name:        "image_title",
				Type:        rpc.ColumnTypeString,
				Description: "The title of the image associated with the item, if any",
			},
			{
				Name:        "categories",
				Type:        rpc.ColumnTypeJSON,
				Description: "A JSON array of categories associated with the item",
			},
			{
				Name:        "enclosures",
				Type:        rpc.ColumnTypeJSON,
				Description: "A JSON array of files associated with the item. Fields: url, length, type",
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

// A destructor to clean up resources
func (t *rssTable) Close() error {
	return nil
}
