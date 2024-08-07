package main

import (
	"context"
	"fmt"

	"github.com/julien040/anyquery/rpc"
)

// A constructor to create a new table instance
// This function is called everytime a new connection is made to the plugin
//
// It should return a new table instance, the database schema and if there is an error
func containerCreator(args rpc.TableCreatorArgs) (rpc.Table, *rpc.DatabaseSchema, error) {
	return &containerTable{}, &rpc.DatabaseSchema{
		Columns: []rpc.DatabaseSchemaColumn{
			{
				Name:        "container_id",
				Type:        rpc.ColumnTypeString,
				IsParameter: true,
				IsRequired:  true,
			},
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
				Name: "created_at",
				Type: rpc.ColumnTypeString,
			},
			{
				Name: "path",
				Type: rpc.ColumnTypeString,
			},
			{
				Name: "args",
				Type: rpc.ColumnTypeString,
			},
			{
				Name: "container_state",
				Type: rpc.ColumnTypeString,
			},
			{
				Name: "image",
				Type: rpc.ColumnTypeString,
			},
			{
				Name: "resolv_conf_path",
				Type: rpc.ColumnTypeString,
			},
			{
				Name: "hostname_path",
				Type: rpc.ColumnTypeString,
			},
			{
				Name: "hosts_path",
				Type: rpc.ColumnTypeString,
			},
			{
				Name: "log_path",
				Type: rpc.ColumnTypeString,
			},
			{
				Name: "name",
				Type: rpc.ColumnTypeString,
			},
			{
				Name: "restart_count",
				Type: rpc.ColumnTypeInt,
			},
			{
				Name: "driver",
				Type: rpc.ColumnTypeString,
			},
			{
				Name: "platform",
				Type: rpc.ColumnTypeString,
			},
			{
				Name: "mount_label",
				Type: rpc.ColumnTypeString,
			},
			{
				Name: "process_label",
				Type: rpc.ColumnTypeString,
			},
			{
				Name: "host_config",
				Type: rpc.ColumnTypeString,
			},
			{
				Name: "mounts",
				Type: rpc.ColumnTypeString,
			},
			{
				Name: "config",
				Type: rpc.ColumnTypeString,
			},
			{
				Name: "network_settings",
				Type: rpc.ColumnTypeString,
			},
		},
	}, nil
}

type containerTable struct {
}

type containerCursor struct {
}

// Return a slice of rows that will be returned to Anyquery and filtered.
// The second return value is true if the cursor has no more rows to return
//
// The constraints are used for optimization purposes to "pre-filter" the rows
// If the rows returned don't match the constraints, it's not an issue. Anyquery will filter them out
func (t *containerCursor) Query(constraints rpc.QueryConstraint) ([][]interface{}, bool, error) {
	client, err := createClient(constraints, 1)
	if err != nil {
		return nil, true, fmt.Errorf("failed to create client: %w", err)
	}

	containerID := retrieveArgString(constraints, 0)
	if containerID == "" {
		return nil, true, fmt.Errorf("missing container ID")
	}

	container, err := client.ContainerInspect(context.Background(), containerID)
	if err != nil {
		return nil, true, fmt.Errorf("failed to inspect container: %w", err)
	}

	return [][]interface{}{
		{
			container.ID,
			container.Created,
			container.Path,
			container.Args,
			serializeJSON(container.State),
			container.Image,
			container.ResolvConfPath,
			container.HostnamePath,
			container.HostsPath,
			container.LogPath,
			container.Name,
			container.RestartCount,
			container.Driver,
			container.Platform,
			container.MountLabel,
			container.ProcessLabel,
			serializeJSON(container.HostConfig),
			serializeJSON(container.Mounts),
			serializeJSON(container.Config),
			serializeJSON(container.NetworkSettings),
		},
	}, true, nil

}

// Create a new cursor that will be used to read rows
func (t *containerTable) CreateReader() rpc.ReaderInterface {
	return &containerCursor{}
}

// A slice of rows to insert
func (t *containerTable) Insert(rows [][]interface{}) error {
	return nil
}

// A slice of rows to update
// The first element of each row is the primary key
// while the rest are the values to update
// The primary key is therefore present twice
func (t *containerTable) Update(rows [][]interface{}) error {
	return nil
}

// A slice of primary keys to delete
func (t *containerTable) Delete(primaryKeys []interface{}) error {
	return nil
}

// A destructor to clean up resources
func (t *containerTable) Close() error {
	return nil
}
