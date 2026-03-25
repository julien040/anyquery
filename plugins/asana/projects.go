package main

import (
	"fmt"

	"github.com/julien040/anyquery/rpc"
)

func projectsCreator(args rpc.TableCreatorArgs) (rpc.Table, *rpc.DatabaseSchema, error) {
	token := args.UserConfig.GetString("token")
	if token == "" {
		return nil, nil, fmt.Errorf("token must be set in the plugin configuration")
	}

	return &projectsTable{
		client: NewAsanaClient(token),
	}, &rpc.DatabaseSchema{
		PrimaryKey: 1,
		Columns: []rpc.DatabaseSchemaColumn{
			{
				Name:        "workspace_gid",
				Type:        rpc.ColumnTypeString,
				Description: "The GID of the workspace to list projects from.",
				IsParameter: true,
				IsRequired:  true,
			},
			{
				Name:        "gid",
				Type:        rpc.ColumnTypeString,
				Description: "Globally unique project identifier.",
			},
			{
				Name:        "name",
				Type:        rpc.ColumnTypeString,
				Description: "The name of the project.",
			},
			{
				Name:        "owner",
				Type:        rpc.ColumnTypeString,
				Description: "Name of the project owner.",
			},
			{
				Name:        "created_at",
				Type:        rpc.ColumnTypeString,
				Description: "Timestamp when the project was created (RFC3339).",
			},
			{
				Name:        "modified_at",
				Type:        rpc.ColumnTypeString,
				Description: "Timestamp when the project was last modified (RFC3339).",
			},
			{
				Name:        "archived",
				Type:        rpc.ColumnTypeBool,
				Description: "Whether the project is archived.",
			},
			{
				Name:        "color",
				Type:        rpc.ColumnTypeString,
				Description: "The color of the project.",
			},
			{
				Name:        "notes",
				Type:        rpc.ColumnTypeString,
				Description: "Free-form notes associated with the project.",
			},
			{
				Name:        "workspace",
				Type:        rpc.ColumnTypeString,
				Description: "Name of the workspace the project belongs to.",
			},
			{
				Name:        "team",
				Type:        rpc.ColumnTypeString,
				Description: "Name of the team the project belongs to.",
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
// 5: modified_at
// 6: archived
// 7: color
// 8: notes
// 9: workspace
// 10: team

type projectsTable struct {
	client *AsanaClient
}

type projectsCursor struct {
	table  *projectsTable
	offset string
}

func (t *projectsTable) CreateReader() rpc.ReaderInterface {
	return &projectsCursor{table: t}
}

func (pc *projectsCursor) Query(constraints rpc.QueryConstraint) ([][]interface{}, bool, error) {
	params := map[string]string{
		"limit":      "100",
		"opt_fields": "gid,name,owner.name,created_at,modified_at,archived,color,notes,workspace.name,team.name",
	}

	workspaceGID := constraints.GetColumnConstraint(0).GetStringValue()
	if workspaceGID == "" {
		return nil, true, fmt.Errorf("workspace_gid is required")
	}
	params["workspace"] = workspaceGID
	if pc.offset != "" {
		params["offset"] = pc.offset
	}

	resp, err := pc.table.client.client.R().
		SetQueryParams(params).
		SetResult(&ProjectsQueryResponse{}).
		Get("/projects")
	if err != nil {
		return nil, true, err
	}
	if resp.IsError() {
		return nil, true, fmt.Errorf("failed to fetch projects (%d): %s", resp.StatusCode(), resp.String())
	}

	result, ok := resp.Result().(*ProjectsQueryResponse)
	if !ok || result == nil {
		return nil, true, fmt.Errorf("unexpected response format")
	}

	rows := make([][]interface{}, len(result.Data))
	for i, p := range result.Data {
		var ownerName interface{}
		if p.Owner != nil {
			ownerName = p.Owner.Name
		}

		rows[i] = []interface{}{
			p.Gid,
			p.Name,
			ownerName,
			nilIfEmpty(p.CreatedAt),
			nilIfEmpty(p.ModifiedAt),
			p.Archived,
			nilIfEmpty(p.Color),
			nilIfEmpty(p.Notes),
			nilIfEmpty(p.Workspace.Name),
			nilIfEmpty(p.Team.Name),
		}
	}

	pc.offset = result.NextPage.Offset
	return rows, pc.offset == "", nil
}

func (t *projectsTable) Close() error {
	return nil
}
