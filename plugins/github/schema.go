package main

import "github.com/julien040/anyquery/rpc"

var repositorySchema = []rpc.DatabaseSchemaColumn{
	{
		Name: "id",
		Type: rpc.ColumnTypeInt,
	},
	{
		Name: "node_id",
		Type: rpc.ColumnTypeString,
	},
	{
		Name: "owner",
		Type: rpc.ColumnTypeString,
	},
	{
		Name: "name",
		Type: rpc.ColumnTypeString,
	},
	{
		Name: "full_name",
		Type: rpc.ColumnTypeString,
	},
	{
		Name: "description",
		Type: rpc.ColumnTypeString,
	},
	{
		Name: "homepage",
		Type: rpc.ColumnTypeString,
	},
	{
		Name: "default_branch",
		Type: rpc.ColumnTypeString,
	},
	{
		Name: "created_at",
		Type: rpc.ColumnTypeString,
	},
	{
		Name: "pushed_at",
		Type: rpc.ColumnTypeString,
	},
	{
		Name: "updated_at",
		Type: rpc.ColumnTypeString,
	},
	{
		Name: "html_url",
		Type: rpc.ColumnTypeString,
	},
	{
		Name: "clone_url",
		Type: rpc.ColumnTypeString,
	},
	{
		Name: "git_url",
		Type: rpc.ColumnTypeString,
	},
	{
		Name: "mirror_url",
		Type: rpc.ColumnTypeString,
	},
	{
		Name: "ssh_url",
		Type: rpc.ColumnTypeString,
	},
	{
		Name: "language",
		Type: rpc.ColumnTypeString,
	},
	{
		Name: "is_fork",
		Type: rpc.ColumnTypeInt,
	},
	{
		Name: "forks_count",
		Type: rpc.ColumnTypeInt,
	},
	{
		Name: "network_count",
		Type: rpc.ColumnTypeInt,
	},
	{
		Name: "open_issues_count",
		Type: rpc.ColumnTypeInt,
	},
	{
		Name: "stargazers_count",
		Type: rpc.ColumnTypeInt,
	},
	{
		Name: "subscribers_count",
		Type: rpc.ColumnTypeInt,
	},
	{
		Name: "size",
		Type: rpc.ColumnTypeInt,
	},
	{
		Name: "allow_rebase_merge",
		Type: rpc.ColumnTypeInt,
	},
	{
		Name: "allow_update_branch",
		Type: rpc.ColumnTypeInt,
	},
	{
		Name: "allow_squash_merge",
		Type: rpc.ColumnTypeInt,
	},
	{
		Name: "allow_merge_commit",
		Type: rpc.ColumnTypeInt,
	},
	{
		Name: "allow_auto_merge",
		Type: rpc.ColumnTypeInt,
	},
	{
		Name: "allow_forking",
		Type: rpc.ColumnTypeInt,
	},
	{
		Name: "web_commit_signoff_required",
		Type: rpc.ColumnTypeInt,
	},
	{
		Name: "delete_branch_on_merge",
		Type: rpc.ColumnTypeInt,
	},
	{
		Name: "topics",
		Type: rpc.ColumnTypeString,
	},
	{
		Name: "custom_properties",
		Type: rpc.ColumnTypeString,
	},
	{
		Name: "archived",
		Type: rpc.ColumnTypeInt,
	},
	{
		Name: "disabled",
		Type: rpc.ColumnTypeInt,
	},
	{
		Name: "visibility",
		Type: rpc.ColumnTypeString,
	},
}

var gistSchema = []rpc.DatabaseSchemaColumn{
	{
		Name: "id",
		Type: rpc.ColumnTypeString,
	},
	{
		Name: "gist_url",
		Type: rpc.ColumnTypeString,
	},
	{
		Name: "by",
		Type: rpc.ColumnTypeString,
	},
	{
		Name: "user_url",
		Type: rpc.ColumnTypeString,
	},
	{
		Name: "description",
		Type: rpc.ColumnTypeString,
	},
	{
		Name: "comments",
		Type: rpc.ColumnTypeInt,
	},
	{
		Name: "public",
		Type: rpc.ColumnTypeInt,
	},
	{
		Name: "created_at",
		Type: rpc.ColumnTypeString,
	},
	{
		Name: "updated_at",
		Type: rpc.ColumnTypeString,
	},
}
