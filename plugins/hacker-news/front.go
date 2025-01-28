package main

import (
	"fmt"

	"github.com/julien040/anyquery/rpc"
)

// A constructor to create a new table instance
// This function is called everytime a new connection is made to the plugin
//
// It should return a new table instance, the database schema and if there is an error
func frontCreator(args rpc.TableCreatorArgs) (rpc.Table, *rpc.DatabaseSchema, error) {
	return &listTable{
			endpoint: "https://hacker-news.firebaseio.com/v0/topstories.json",
		}, &rpc.DatabaseSchema{
			Columns: []rpc.DatabaseSchemaColumn{
				{
					Name:        "id",
					Type:        rpc.ColumnTypeInt,
					Description: "The ID of the Hacker News item",
				},
			},
		}, nil
}

func newCreator(args rpc.TableCreatorArgs) (rpc.Table, *rpc.DatabaseSchema, error) {
	return &listTable{
			endpoint: "https://hacker-news.firebaseio.com/v0/newstories.json",
		}, &rpc.DatabaseSchema{
			Columns: []rpc.DatabaseSchemaColumn{
				{
					Name: "id",
					Type: rpc.ColumnTypeInt,
				},
			},
		}, nil
}

func bestCreator(args rpc.TableCreatorArgs) (rpc.Table, *rpc.DatabaseSchema, error) {
	return &listTable{
			endpoint: "https://hacker-news.firebaseio.com/v0/beststories.json",
		}, &rpc.DatabaseSchema{
			Columns: []rpc.DatabaseSchemaColumn{
				{
					Name: "id",
					Type: rpc.ColumnTypeInt,
				},
			},
		}, nil
}

func askCreator(args rpc.TableCreatorArgs) (rpc.Table, *rpc.DatabaseSchema, error) {
	return &listTable{
			endpoint: "https://hacker-news.firebaseio.com/v0/askstories.json",
		}, &rpc.DatabaseSchema{
			Columns: []rpc.DatabaseSchemaColumn{
				{
					Name: "id",
					Type: rpc.ColumnTypeInt,
				},
			},
		}, nil
}

func showCreator(args rpc.TableCreatorArgs) (rpc.Table, *rpc.DatabaseSchema, error) {
	return &listTable{
			endpoint: "https://hacker-news.firebaseio.com/v0/showstories.json",
		}, &rpc.DatabaseSchema{
			Columns: []rpc.DatabaseSchemaColumn{
				{
					Name: "id",
					Type: rpc.ColumnTypeInt,
				},
			},
		}, nil
}

func jobsCreator(args rpc.TableCreatorArgs) (rpc.Table, *rpc.DatabaseSchema, error) {
	return &listTable{
			endpoint: "https://hacker-news.firebaseio.com/v0/jobstories.json",
		}, &rpc.DatabaseSchema{
			Columns: []rpc.DatabaseSchemaColumn{
				{
					Name: "id",
					Type: rpc.ColumnTypeInt,
				},
			},
		}, nil
}

type ApiResponse []int64

type listTable struct {
	endpoint string
}

type listCursor struct {
	endpoint string
}

// Return a slice of rows that will be returned to Anyquery and filtered.
// The second return value is true if the cursor has no more rows to return
//
// The constraints are used for optimization purposes to "pre-filter" the rows
// If the rows returned don't match the constraints, it's not an issue. Anyquery will filter them out
func (t *listCursor) Query(constraints rpc.QueryConstraint) ([][]interface{}, bool, error) {

	data := ApiResponse{}

	// Fetch the data from the API
	res, err := client.R().SetResult(&data).Get(t.endpoint)
	if err != nil {
		return nil, true, fmt.Errorf("failed to fetch data from the API: %w", err)
	}

	if res.IsError() {
		return nil, true, fmt.Errorf("failed to fetch data from the API(%d): %s", res.StatusCode(), res.String())
	}

	// Convert the data to a slice of rows
	rows := make([][]interface{}, len(data))
	for i, id := range data {
		rows[i] = []interface{}{id}
	}

	return rows, true, nil
}

// Create a new cursor that will be used to read rows
func (t *listTable) CreateReader() rpc.ReaderInterface {
	return &listCursor{
		endpoint: t.endpoint,
	}
}

// A destructor to clean up resources
func (t *listTable) Close() error {
	return nil
}
