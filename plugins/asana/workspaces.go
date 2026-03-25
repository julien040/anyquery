package main

import (
	"fmt"

	"github.com/julien040/anyquery/rpc"
)

func workspacesCreator(args rpc.TableCreatorArgs) (rpc.Table, *rpc.DatabaseSchema, error) {
	token := args.UserConfig.GetString("token")
	if token == "" {
		return nil, nil, fmt.Errorf("token must be set in the plugin configuration")
	}

	return &workspacesTable{
		client: NewAsanaClient(token),
	}, &rpc.DatabaseSchema{
		PrimaryKey: 0,
		Columns: []rpc.DatabaseSchemaColumn{
			{
				Name:        "gid",
				Type:        rpc.ColumnTypeString,
				Description: "Globally unique workspace identifier.",
			},
			{
				Name:        "name",
				Type:        rpc.ColumnTypeString,
				Description: "The name of the workspace.",
			},
			{
				Name:        "is_organization",
				Type:        rpc.ColumnTypeInt,
				Description: "1 if the workspace is an organization, 0 otherwise.",
			},
		},
	}, nil
}

type workspacesTable struct {
	client *AsanaClient
}

type workspacesCursor struct {
	table  *workspacesTable
	offset string
}

func (t *workspacesTable) CreateReader() rpc.ReaderInterface {
	return &workspacesCursor{table: t}
}

func (wc *workspacesCursor) Query(constraints rpc.QueryConstraint) ([][]interface{}, bool, error) {
	params := map[string]string{
		"limit":      "100",
		"opt_fields": "gid,name,is_organization",
	}
	if wc.offset != "" {
		params["offset"] = wc.offset
	}

	resp, err := wc.table.client.client.R().
		SetQueryParams(params).
		SetResult(&WorkspacesQueryResponse{}).
		Get("/workspaces")
	if err != nil {
		return nil, true, err
	}
	if resp.IsError() {
		return nil, true, fmt.Errorf("failed to fetch workspaces (%d): %s", resp.StatusCode(), resp.String())
	}

	result, ok := resp.Result().(*WorkspacesQueryResponse)
	if !ok || result == nil {
		return nil, true, fmt.Errorf("unexpected response format")
	}

	rows := make([][]interface{}, len(result.Data))
	for i, w := range result.Data {
		var isOrg int64
		if w.IsOrganization {
			isOrg = 1
		}
		rows[i] = []interface{}{
			w.Gid,
			w.Name,
			isOrg,
		}
	}

	wc.offset = result.NextPage.Offset
	return rows, wc.offset == "", nil
}

func (t *workspacesTable) Close() error {
	return nil
}
