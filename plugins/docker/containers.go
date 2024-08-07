package main

import (
	"context"
	"fmt"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/julien040/anyquery/rpc"
)

// A constructor to create a new table instance
// This function is called everytime a new connection is made to the plugin
//
// It should return a new table instance, the database schema and if there is an error
func containersCreator(args rpc.TableCreatorArgs) (rpc.Table, *rpc.DatabaseSchema, error) {
	return &containersTable{}, &rpc.DatabaseSchema{
		Columns: []rpc.DatabaseSchemaColumn{
			{
				Name:        "host",
				Type:        rpc.ColumnTypeString,
				IsParameter: true,
			},
			{
				Name: "id",
				Type: rpc.ColumnTypeString,
			},
			{
				Name: "names",
				Type: rpc.ColumnTypeString,
			},
			{
				Name: "image",
				Type: rpc.ColumnTypeString,
			},
			{
				Name: "image_id",
				Type: rpc.ColumnTypeString,
			},
			{
				Name: "command",
				Type: rpc.ColumnTypeString,
			},
			{
				Name: "created_at",
				Type: rpc.ColumnTypeString,
			},
			{
				Name: "ports",
				Type: rpc.ColumnTypeString,
			},
			{
				Name: "labels",
				Type: rpc.ColumnTypeString,
			},
			{
				Name: "size_rw",
				Type: rpc.ColumnTypeInt,
			},
			{
				Name: "size_root_fs",
				Type: rpc.ColumnTypeInt,
			},
			{
				Name: "state",
				Type: rpc.ColumnTypeString,
			},
			{
				Name: "status",
				Type: rpc.ColumnTypeString,
			},
			{
				Name: "networks",
				Type: rpc.ColumnTypeString,
			},
			{
				Name: "mounts",
				Type: rpc.ColumnTypeString,
			},
		},
	}, nil
}

type containersTable struct {
}

type containersCursor struct {
}

// Return a slice of rows that will be returned to Anyquery and filtered.
// The second return value is true if the cursor has no more rows to return
//
// The constraints are used for optimization purposes to "pre-filter" the rows
// If the rows returned don't match the constraints, it's not an issue. Anyquery will filter them out
func (t *containersCursor) Query(constraints rpc.QueryConstraint) ([][]interface{}, bool, error) {
	// Extract the host from the constraints
	client, err := createClient(constraints, 0)
	if err != nil {
		return nil, true, fmt.Errorf("failed to create docker client: %w", err)
	}

	containers, err := client.ContainerList(context.Background(), container.ListOptions{
		All:  true,
		Size: true,
	})
	if err != nil {
		return nil, true, fmt.Errorf("failed to list containers: %w", err)
	}

	rows := [][]interface{}{}
	for _, container := range containers {
		rows = append(rows, []interface{}{
			container.ID,
			serializeJSON(container.Names),
			container.Image,
			container.ImageID,
			container.Command,
			time.Unix(container.Created, 0).Format(time.RFC3339),
			serializeJSON(container.Ports),
			serializeJSON(container.Labels),
			container.SizeRw,
			container.SizeRootFs,
			container.State,
			container.Status,
			serializeJSON(container.NetworkSettings.Networks),
			serializeJSON(container.Mounts),
		})
	}

	return rows, true, nil
}

// Create a new cursor that will be used to read rows
func (t *containersTable) CreateReader() rpc.ReaderInterface {
	return &containersCursor{}
}

// A slice of rows to insert
func (t *containersTable) Insert(rows [][]interface{}) error {
	return nil
}

// A slice of rows to update
// The first element of each row is the primary key
// while the rest are the values to update
// The primary key is therefore present twice
func (t *containersTable) Update(rows [][]interface{}) error {
	return nil
}

// A slice of primary keys to delete
func (t *containersTable) Delete(primaryKeys []interface{}) error {
	return nil
}

// A destructor to clean up resources
func (t *containersTable) Close() error {
	return nil
}
