package main

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"log"
	"time"

	"github.com/dgraph-io/badger/v4"
	"github.com/julien040/anyquery/rpc"
)

// A constructor to create a new table instance
// This function is called everytime a new connection is made to the plugin
//
// It should return a new table instance, the database schema and if there is an error
func projectsCreator(args rpc.TableCreatorArgs) (rpc.Table, *rpc.DatabaseSchema, error) {
	token, err := getToken(args.UserConfig)
	if err != nil {
		return nil, nil, err
	}

	db, err := openDatabase("projects", token)
	if err != nil {
		return nil, nil, err
	}

	return &projectsTable{
			token: token,
			db:    db,
		}, &rpc.DatabaseSchema{
			Columns: []rpc.DatabaseSchemaColumn{
				{
					Name:        "team_id",
					Type:        rpc.ColumnTypeString,
					IsParameter: true,
				},
				{
					Name: "account_id",
					Type: rpc.ColumnTypeString,
				},
				{
					Name: "created_at",
					Type: rpc.ColumnTypeString,
				},
				{
					Name: "updated_at",
					Type: rpc.ColumnTypeString,
				},
				{
					Name: "framework",
					Type: rpc.ColumnTypeString,
				},
				{
					Name: "project_id",
					Type: rpc.ColumnTypeString,
				},
				{
					Name: "name",
					Type: rpc.ColumnTypeString,
				},
				{
					Name: "node_version",
					Type: rpc.ColumnTypeString,
				},
				{
					Name: "serverless_region",
					Type: rpc.ColumnTypeString,
				},
			},
		}, nil
}

type projectsTable struct {
	token string
	db    *badger.DB
}

type projectsCursor struct {
	db    *badger.DB
	token string
	next  int64
}

// Return a slice of rows that will be returned to Anyquery and filtered.
// The second return value is true if the cursor has no more rows to return
//
// The constraints are used for optimization purposes to "pre-filter" the rows
// If the rows returned don't match the constraints, it's not an issue. Anyquery will filter them out
func (t *projectsCursor) Query(constraints rpc.QueryConstraint) ([][]interface{}, bool, error) {
	teamID := retrieveArgString(constraints, 0)

	// Retrieve the projects
	cacheKey := fmt.Sprintf("projects-%s-%d", teamID, t.next)
	rows := [][]interface{}{}
	apiResponse := &Projects{}

	// Try to load the cache
	err := t.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte(cacheKey))
		if err != nil {
			return err
		}

		return item.Value(func(val []byte) error {
			dec := gob.NewDecoder(bytes.NewReader(val))
			return dec.Decode(apiResponse)
		})
	})
	if err != nil {
		endpoint := "https://api.vercel.com/v9/projects"
		req := client.R().SetHeader("Authorization", "Bearer "+t.token).SetResult(apiResponse).
			SetQueryParam("teamId", teamID).
			SetQueryParam("limit", fmt.Sprintf("%d", EntriesPerPage))
		if t.next != 0 {
			req.SetQueryParam("from", fmt.Sprintf("%d", t.next))
		}
		res, err := req.Get(endpoint)
		if err != nil {
			return nil, true, fmt.Errorf("failed to fetch projects: %v", err)
		}
		if res.IsError() {
			return nil, true, fmt.Errorf("failed to fetch projects(code %s): text %s", res.Status(), res.String())
		}

		// Save the cache
		err = t.db.Update(func(txn *badger.Txn) error {
			buf := bytes.Buffer{}
			enc := gob.NewEncoder(&buf)
			err := enc.Encode(apiResponse)
			if err != nil {
				return err
			}

			e := badger.NewEntry([]byte(cacheKey), buf.Bytes()).WithTTL(ttl)
			return txn.SetEntry(e)
		})
		if err != nil {
			log.Printf("Failed to save cache: %v", err)
		}

	}

	// Update the next page
	if apiResponse.Pagination.Next == nil {
		t.next = 0
	} else {
		t.next = int64(apiResponse.Pagination.Next.(float64))
	}

	// Convert the projects to rows
	for _, project := range apiResponse.Projects {
		region := interface{}(nil)
		if project.ServerlessFunctionRegion != nil {
			region = *project.ServerlessFunctionRegion
		}
		framework := interface{}(nil)
		if project.Framework != nil {
			framework = *project.Framework
		}
		rows = append(rows, []interface{}{
			string(project.AccountID),
			time.Unix(project.CreatedAt, 0).Format(time.RFC3339),
			time.Unix(project.UpdatedAt, 0).Format(time.RFC3339),
			framework,
			string(project.ID),
			project.Name,
			project.NodeVersion,
			region,
		})
	}

	return rows, len(rows) < EntriesPerPage || t.next == 0, nil
}

// Create a new cursor that will be used to read rows
func (t *projectsTable) CreateReader() rpc.ReaderInterface {
	return &projectsCursor{
		db:    t.db,
		token: t.token,
		next:  0,
	}
}

// A slice of rows to insert
func (t *projectsTable) Insert(rows [][]interface{}) error {
	return nil
}

// A slice of rows to update
// The first element of each row is the primary key
// while the rest are the values to update
// The primary key is therefore present twice
func (t *projectsTable) Update(rows [][]interface{}) error {
	return nil
}

// A slice of primary keys to delete
func (t *projectsTable) Delete(primaryKeys []interface{}) error {
	return nil
}

// A destructor to clean up resources
func (t *projectsTable) Close() error {
	return nil
}
