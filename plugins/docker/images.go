package main

import (
	"context"
	"fmt"
	"time"

	"github.com/docker/docker/api/types/image"
	"github.com/julien040/anyquery/rpc"
)

// A constructor to create a new table instance
// This function is called everytime a new connection is made to the plugin
//
// It should return a new table instance, the database schema and if there is an error
func imagesCreator(args rpc.TableCreatorArgs) (rpc.Table, *rpc.DatabaseSchema, error) {
	return &imagesTable{}, &rpc.DatabaseSchema{
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
				Description: "The ID of the image",
			},
			{
				Name:        "created_at",
				Type:        rpc.ColumnTypeDateTime,
				Description: "The time the image was created (RFC3339)",
			},
			{
				Name:        "labels",
				Type:        rpc.ColumnTypeJSON,
				Description: "A JSON object of labels assigned to the image",
			},
			{
				Name:        "parent_id",
				Type:        rpc.ColumnTypeString,
				Description: "The ID of the parent image",
			},
			{
				Name:        "repo_tags",
				Type:        rpc.ColumnTypeString,
				Description: "A JSON array of tags assigned to the image",
			},
			{
				Name:        "repo_digests",
				Type:        rpc.ColumnTypeString,
				Description: "A JSON array of digests assigned to the image",
			},
			{
				Name:        "container_count",
				Type:        rpc.ColumnTypeInt,
				Description: "The number of containers using the image",
			},
			{
				Name:        "shared_size",
				Type:        rpc.ColumnTypeInt,
				Description: "The size of the image shared with other images",
			},
			{
				Name:        "size",
				Type:        rpc.ColumnTypeInt,
				Description: "The size of the image",
			},
		},
	}, nil
}

type imagesTable struct {
}

type imagesCursor struct {
}

// Return a slice of rows that will be returned to Anyquery and filtered.
// The second return value is true if the cursor has no more rows to return
//
// The constraints are used for optimization purposes to "pre-filter" the rows
// If the rows returned don't match the constraints, it's not an issue. Anyquery will filter them out
func (t *imagesCursor) Query(constraints rpc.QueryConstraint) ([][]interface{}, bool, error) {
	client, err := createClient(constraints, 0)
	if err != nil {
		return nil, true, fmt.Errorf("failed to create client: %w", err)
	}

	images, err := client.ImageList(context.Background(), image.ListOptions{
		All:            true,
		SharedSize:     true,
		ContainerCount: true,
	})

	if err != nil {
		return nil, true, fmt.Errorf("failed to list images: %w", err)
	}

	rows := [][]interface{}{}
	for _, img := range images {
		rows = append(rows, []interface{}{
			img.ID,
			time.Unix(img.Created, 0).Format(time.RFC3339),
			serializeJSON(img.Labels),
			img.ParentID,
			img.RepoTags,
			img.RepoDigests,
			img.Containers,
			img.SharedSize,
			img.Size,
		})
	}

	return rows, true, nil
}

// Create a new cursor that will be used to read rows
func (t *imagesTable) CreateReader() rpc.ReaderInterface {
	return &imagesCursor{}
}

// A destructor to clean up resources
func (t *imagesTable) Close() error {
	return nil
}
