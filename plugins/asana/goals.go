package main

import (
	"fmt"

	"github.com/julien040/anyquery/rpc"
)

func goalsCreator(args rpc.TableCreatorArgs) (rpc.Table, *rpc.DatabaseSchema, error) {
	token := args.UserConfig.GetString("token")
	if token == "" {
		return nil, nil, fmt.Errorf("token must be set in the plugin configuration")
	}

	return &goalsTable{
		client: NewAsanaClient(token),
	}, &rpc.DatabaseSchema{
		PrimaryKey: 1,
		Columns: []rpc.DatabaseSchemaColumn{
			{
				Name:        "workspace_gid",
				Type:        rpc.ColumnTypeString,
				Description: "Filter goals by workspace GID.",
				IsParameter: true,
			},
			{
				Name:        "gid",
				Type:        rpc.ColumnTypeString,
				Description: "Globally unique goal identifier.",
			},
			{
				Name:        "name",
				Type:        rpc.ColumnTypeString,
				Description: "The name of the goal.",
			},
			{
				Name:        "owner",
				Type:        rpc.ColumnTypeString,
				Description: "Name of the goal owner.",
			},
			{
				Name:        "created_at",
				Type:        rpc.ColumnTypeString,
				Description: "Timestamp when the goal was created (RFC3339).",
			},
			{
				Name:        "due_on",
				Type:        rpc.ColumnTypeString,
				Description: "Due date of the goal (YYYY-MM-DD).",
			},
			{
				Name:        "status",
				Type:        rpc.ColumnTypeString,
				Description: "Current status of the goal (e.g. on_track, at_risk, missed).",
			},
			{
				Name:        "notes",
				Type:        rpc.ColumnTypeString,
				Description: "Free-form notes associated with the goal.",
			},
			{
				Name:        "workspace",
				Type:        rpc.ColumnTypeString,
				Description: "Name of the workspace the goal belongs to.",
			},
		},
	}, nil
}

// Schema column indices:
// 0: workspace_gid (parameter)
// 1: gid (PK)
// 2: name
// 3: owner
// 4: created_at
// 5: due_on
// 6: status
// 7: notes
// 8: workspace

type goalsTable struct {
	client *AsanaClient
}

type goalsCursor struct {
	table  *goalsTable
	offset string
}

func (t *goalsTable) CreateReader() rpc.ReaderInterface {
	return &goalsCursor{table: t}
}

func (gc *goalsCursor) Query(constraints rpc.QueryConstraint) ([][]interface{}, bool, error) {
	params := map[string]string{
		"limit":      "100",
		"opt_fields": "gid,name,owner.name,created_at,due_on,status,notes,workspace.name",
	}

	workspaceGID := constraints.GetColumnConstraint(0).GetStringValue()
	if workspaceGID != "" {
		params["workspace"] = workspaceGID
	}
	if gc.offset != "" {
		params["offset"] = gc.offset
	}

	resp, err := gc.table.client.client.R().
		SetQueryParams(params).
		SetResult(&GoalsQueryResponse{}).
		Get("/goals")
	if err != nil {
		return nil, true, err
	}
	if resp.IsError() {
		return nil, true, fmt.Errorf("failed to fetch goals (%d): %s", resp.StatusCode(), resp.String())
	}

	result, ok := resp.Result().(*GoalsQueryResponse)
	if !ok || result == nil {
		return nil, true, fmt.Errorf("unexpected response format")
	}

	rows := make([][]interface{}, len(result.Data))
	for i, g := range result.Data {
		var ownerName interface{}
		if g.Owner != nil {
			ownerName = g.Owner.Name
		}

		rows[i] = []interface{}{
			g.Gid,
			g.Name,
			ownerName,
			nilIfEmpty(g.CreatedAt),
			nilIfEmpty(g.DueOn),
			nilIfEmpty(g.Status),
			nilIfEmpty(g.Notes),
			nilIfEmpty(g.Workspace.Name),
		}
	}

	gc.offset = result.NextPage.Offset
	return rows, gc.offset == "", nil
}

func (t *goalsTable) Close() error {
	return nil
}
