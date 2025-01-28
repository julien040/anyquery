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
				Description: "The ID of the container. Can be retrieved from the containers table, or by running `docker ps`",
			},
			{
				Name:        "host",
				Type:        rpc.ColumnTypeString,
				IsParameter: true,
				Description: "The Docker host to connect to. Can be a hostname or an IP address. Defaults to `unix:///var/run/docker.sock`",
			},

			{
				Name:        "id",
				Type:        rpc.ColumnTypeString,
				Description: "The ID of the container",
			},
			{
				Name:        "created_at",
				Type:        rpc.ColumnTypeDateTime,
				Description: "The time the container was created (RFC3339)",
			},
			{
				Name:        "path",
				Type:        rpc.ColumnTypeString,
				Description: "The path to the command that is running in the container",
			},
			{
				Name:        "args",
				Type:        rpc.ColumnTypeString,
				Description: "The arguments to the command that is running in the container",
			},
			{
				Name:        "container_state",
				Type:        rpc.ColumnTypeString,
				Description: "The state of the container. Can be `created`, `restarting`, `running`, `removing`, `paused`, `exited`, `dead`",
			},
			{
				Name:        "image",
				Type:        rpc.ColumnTypeString,
				Description: "The image used to create the container",
			},
			{
				Name:        "resolv_conf_path",
				Type:        rpc.ColumnTypeString,
				Description: "The path to the resolv.conf file used by the container",
			},
			{
				Name:        "hostname_path",
				Type:        rpc.ColumnTypeString,
				Description: "The path to the hostname file used by the container",
			},
			{
				Name:        "hosts_path",
				Type:        rpc.ColumnTypeString,
				Description: "The path to the hosts file used by the container",
			},
			{
				Name:        "log_path",
				Type:        rpc.ColumnTypeString,
				Description: "The path to the log file used by the container",
			},
			{
				Name:        "name",
				Type:        rpc.ColumnTypeString,
				Description: "The name of the container",
			},
			{
				Name:        "restart_count",
				Type:        rpc.ColumnTypeInt,
				Description: "The number of times the container has been restarted",
			},
			{
				Name:        "driver",
				Type:        rpc.ColumnTypeString,
				Description: "The driver used by the container",
			},
			{
				Name:        "platform",
				Type:        rpc.ColumnTypeString,
				Description: "The platform of the container",
			},
			{
				Name:        "mount_label",
				Type:        rpc.ColumnTypeString,
				Description: "The mount label of the container",
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

// A destructor to clean up resources
func (t *containerTable) Close() error {
	return nil
}
