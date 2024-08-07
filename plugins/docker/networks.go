package main

import (
	"context"
	"fmt"
	"time"

	"github.com/docker/docker/api/types/network"
	"github.com/julien040/anyquery/rpc"
)

// A constructor to create a new table instance
// This function is called everytime a new connection is made to the plugin
//
// It should return a new table instance, the database schema and if there is an error
func networksCreator(args rpc.TableCreatorArgs) (rpc.Table, *rpc.DatabaseSchema, error) {
	return &networksTable{}, &rpc.DatabaseSchema{
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
				Name: "name",
				Type: rpc.ColumnTypeString,
			},
			{
				Name: "created_at",
				Type: rpc.ColumnTypeString,
			},
			{
				Name: "scope",
				Type: rpc.ColumnTypeString,
			},
			{
				Name: "driver",
				Type: rpc.ColumnTypeString,
			},
			{
				Name: "enable_ipv6",
				Type: rpc.ColumnTypeBool,
			},
			{
				Name: "ipam",
				Type: rpc.ColumnTypeString,
			},
			{
				Name: "containers",
				Type: rpc.ColumnTypeString,
			},
			{
				Name: "options",
				Type: rpc.ColumnTypeString,
			},
			{
				Name: "labels",
				Type: rpc.ColumnTypeString,
			},
			{
				Name: "peers",
				Type: rpc.ColumnTypeString,
			},
			{
				Name: "services",
				Type: rpc.ColumnTypeString,
			},
			{
				Name: "internal",
				Type: rpc.ColumnTypeBool,
			},
			{
				Name: "attachable",
				Type: rpc.ColumnTypeBool,
			},
			{
				Name: "ingress",
				Type: rpc.ColumnTypeBool,
			},
			{
				Name: "config_only",
				Type: rpc.ColumnTypeBool,
			},
			{
				Name: "config_from",
				Type: rpc.ColumnTypeString,
			},
		},
	}, nil
}

type networksTable struct {
}

type networksCursor struct {
}

// Return a slice of rows that will be returned to Anyquery and filtered.
// The second return value is true if the cursor has no more rows to return
//
// The constraints are used for optimization purposes to "pre-filter" the rows
// If the rows returned don't match the constraints, it's not an issue. Anyquery will filter them out
func (t *networksCursor) Query(constraints rpc.QueryConstraint) ([][]interface{}, bool, error) {
	client, err := createClient(constraints, 0)
	if err != nil {
		return nil, true, fmt.Errorf("failed to create client: %w", err)
	}

	networks, err := client.NetworkList(context.Background(), network.ListOptions{})
	if err != nil {
		return nil, true, fmt.Errorf("failed to get networks: %w", err)
	}

	rows := [][]interface{}{}
	for _, network := range networks {
		rows = append(rows, []interface{}{
			network.ID,
			network.Name,
			network.Created.Format(time.RFC3339),
			network.Scope,
			network.Driver,
			network.EnableIPv6,
			serializeJSON(network.IPAM),
			serializeJSON(network.Containers),
			serializeJSON(network.Options),
			serializeJSON(network.Labels),
			serializeJSON(network.Peers),
			serializeJSON(network.Services),
			network.Internal,
			network.Attachable,
			network.Ingress,
			network.ConfigOnly,
			serializeJSON(network.ConfigFrom),
		})
	}

	return rows, true, nil
}

// Create a new cursor that will be used to read rows
func (t *networksTable) CreateReader() rpc.ReaderInterface {
	return &networksCursor{}
}

// A slice of rows to insert
func (t *networksTable) Insert(rows [][]interface{}) error {
	return nil
}

// A slice of rows to update
// The first element of each row is the primary key
// while the rest are the values to update
// The primary key is therefore present twice
func (t *networksTable) Update(rows [][]interface{}) error {
	return nil
}

// A slice of primary keys to delete
func (t *networksTable) Delete(primaryKeys []interface{}) error {
	return nil
}

// A destructor to clean up resources
func (t *networksTable) Close() error {
	return nil
}
