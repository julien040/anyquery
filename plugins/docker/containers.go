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
				Description: "The Docker host to connect to. Can be a hostname or an IP address. Defaults to `unix:///var/run/docker.sock` if not set",
			},
			{
				Name:        "id",
				Type:        rpc.ColumnTypeString,
				Description: "The ID of the container",
			},
			{
				Name:        "names",
				Type:        rpc.ColumnTypeJSON,
				Description: "A JSON array of names assigned to the container",
			},
			{
				Name:        "image",
				Type:        rpc.ColumnTypeString,
				Description: "The image used to create the container",
			},
			{
				Name:        "image_id",
				Type:        rpc.ColumnTypeString,
				Description: "The ID of the image used to create the container",
			},
			{
				Name:        "command",
				Type:        rpc.ColumnTypeString,
				Description: "The command that is running in the container",
			},
			{
				Name:        "created_at",
				Type:        rpc.ColumnTypeDateTime,
				Description: "The time the container was created (RFC3339)",
			},
			{
				Name:        "ports",
				Type:        rpc.ColumnTypeJSON,
				Description: "The ports exposed by the container",
			},
			{
				Name:        "labels",
				Type:        rpc.ColumnTypeJSON,
				Description: "A JSON object of labels assigned to the container",
			},
			{
				Name:        "size_rw",
				Type:        rpc.ColumnTypeInt,
				Description: "The size of the RW layer of the container",
			},
			{
				Name:        "size_root_fs",
				Type:        rpc.ColumnTypeInt,
				Description: "The size of the root filesystem of the container",
			},
			{
				Name:        "state",
				Type:        rpc.ColumnTypeString,
				Description: "The state of the container. Can be `created`, `restarting`, `running`, `removing`, `paused`, `exited`, `dead`",
			},
			{
				Name:        "status",
				Type:        rpc.ColumnTypeString,
				Description: "The status of the container",
			},
			{
				Name:        "networks",
				Type:        rpc.ColumnTypeString,
				Description: "A JSON object of networks assigned to the container",
			},
			{
				Name:        "mounts",
				Type:        rpc.ColumnTypeString,
				Description: "A JSON array of mounts assigned to the container",
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

// A destructor to clean up resources
func (t *containersTable) Close() error {
	return nil
}
