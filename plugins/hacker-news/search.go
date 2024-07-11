package main

import (
	"fmt"
	"strconv"

	"github.com/julien040/anyquery/rpc"
)

// A constructor to create a new table instance
// This function is called everytime a new connection is made to the plugin
//
// It should return a new table instance, the database schema and if there is an error
func searchCreator(args rpc.TableCreatorArgs) (rpc.Table, *rpc.DatabaseSchema, error) {
	return &hacker_newsTable{}, &rpc.DatabaseSchema{
		HandlesInsert: false,
		HandlesUpdate: false,
		HandlesDelete: false,
		HandleOffset:  false,
		Columns: []rpc.DatabaseSchemaColumn{
			{
				Name: "id",
				Type: rpc.ColumnTypeString,
			},
			{
				Name: "title",
				Type: rpc.ColumnTypeString,
			},
			{
				Name: "created_at",
				Type: rpc.ColumnTypeString,
			},
			{
				Name: "type",
				Type: rpc.ColumnTypeString,
			},
			{
				Name: "url",
				Type: rpc.ColumnTypeString,
			},
			{
				Name: "author",
				Type: rpc.ColumnTypeString,
			},
			{
				Name: "points",
				Type: rpc.ColumnTypeInt,
			},
			{
				Name: "num_comments",
				Type: rpc.ColumnTypeInt,
			},
			{
				Name: "story_id",
				Type: rpc.ColumnTypeInt,
			},
			{
				Name: "story_title",
				Type: rpc.ColumnTypeString,
			},
			{
				Name: "comment_text",
				Type: rpc.ColumnTypeString,
			},
			{
				Name: "parent_id",
				Type: rpc.ColumnTypeInt,
			},
			{
				Name: "tags",
				Type: rpc.ColumnTypeString,
			},
			{
				Name:        "query",
				Type:        rpc.ColumnTypeString,
				IsParameter: true,
			},
		},
	}, nil
}

type hacker_newsTable struct {
}

type hacker_newsCursor struct {
	pageID          int
	cursorExhausted bool
}

// Return a slice of rows that will be returned to Anyquery and filtered.
// The second return value is true if the cursor has no more rows to return
//
// The constraints are used for optimization purposes to "pre-filter" the rows
// If the rows returned don't match the constraints, it's not an issue. Anyquery will filter them out
func (t *hacker_newsCursor) Query(constraints rpc.QueryConstraint) ([][]interface{}, bool, error) {
	// Check the type constraint and update the endpoint accordingly
	tags := "(story, poll, job, comment, ask_hn, show_hn)"
	tagsPrefix := "" // To add the author tag
	for _, constraint := range constraints.Columns {
		if constraint.ColumnID == 3 {
			if parseStr, ok := constraint.Value.(string); ok {
				tags = parseStr
			}
		} else if constraint.ColumnID == 5 {
			if parseStr, ok := constraint.Value.(string); ok {
				tagsPrefix = fmt.Sprintf(",author_%s", parseStr)
			}
		}
	}

	tags += tagsPrefix

	// If the query is sorted by created_at descending, we use another endpoint
	endpoint := "http://hn.algolia.com/api/v1/search"
	for _, sort := range constraints.OrderBy {
		if sort.ColumnID == 2 && sort.Descending {
			endpoint = "http://hn.algolia.com/api/v1/search_by_date"
			break
		}
	}

	query := ""
	for _, constraint := range constraints.Columns {
		if constraint.ColumnID == 13 {
			if parseStr, ok := constraint.Value.(string); ok {
				query = parseStr
			}
		}
	}

	algoliaRes := HackerNewsAPIResponseAlgolia{}

	// Fetch the data
	res, err := client.R().SetResult(&algoliaRes).SetQueryParam("tags", tags).SetQueryParam("query", query).
		SetQueryParam("page", strconv.Itoa(t.pageID)).SetQueryParam("hitsPerPage", "200").Get(endpoint)
	if err != nil {
		return nil, true, err
	}

	// Check if the request was successful
	if res.IsError() {
		return nil, true, res.Error().(error)
	}

	// Check if we have more pages to fetch
	t.pageID++
	if t.pageID >= algoliaRes.NbPages {
		t.cursorExhausted = true
	}

	// Convert the data to the format expected by Anyquery
	rows := [][]interface{}{}

	for _, hit := range algoliaRes.Hits {
		// Get the type
		Type := ""
		for _, tag := range hit.Tags {
			if tag == "story" || tag == "poll" || tag == "job" || tag == "comment" {
				Type = tag
				break
			}
		}

		rows = append(rows, []interface{}{
			hit.ObjectID,
			hit.Title,
			hit.CreatedAt,
			Type,
			hit.URL,
			hit.Author,
			hit.Points,
			hit.NumComments,
			hit.StoryID,
			hit.StoryTitle,
			hit.CommentText,
			hit.ParentID,
			hit.Tags,
		})
	}

	return rows, t.cursorExhausted, nil
}

// Create a new cursor that will be used to read rows
func (t *hacker_newsTable) CreateReader() rpc.ReaderInterface {
	return &hacker_newsCursor{}
}

// A slice of rows to insert
func (t *hacker_newsTable) Insert(rows [][]interface{}) error {
	return nil
}

// A slice of rows to update
// The first element of each row is the primary key
// while the rest are the values to update
// The primary key is therefore present twice
func (t *hacker_newsTable) Update(rows [][]interface{}) error {
	return nil
}

// A slice of primary keys to delete
func (t *hacker_newsTable) Delete(primaryKeys []interface{}) error {
	return nil
}

// A destructor to clean up resources
func (t *hacker_newsTable) Close() error {
	return nil
}
