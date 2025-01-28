package main

import "github.com/julien040/anyquery/rpc"

var repositorySchema = []rpc.DatabaseSchemaColumn{
	{
		Name:        "id",
		Type:        rpc.ColumnTypeInt,
		Description: "The ID of the repository",
	},
	{
		Name:        "node_id",
		Type:        rpc.ColumnTypeString,
		Description: "The node ID of the repository",
	},
	{
		Name:        "owner",
		Type:        rpc.ColumnTypeString,
		Description: "The username of the owner of the repository (or the organization)",
	},
	{
		Name:        "name",
		Type:        rpc.ColumnTypeString,
		Description: "The name of the repository",
	},
	{
		Name:        "full_name",
		Type:        rpc.ColumnTypeString,
		Description: "The full name of the repository (owner/name)",
	},
	{
		Name:        "description",
		Type:        rpc.ColumnTypeString,
		Description: "A short description of the repository",
	},
	{
		Name:        "homepage",
		Type:        rpc.ColumnTypeString,
		Description: "A link the repository has set as its homepage",
	},
	{
		Name:        "default_branch",
		Type:        rpc.ColumnTypeString,
		Description: "The default branch of the repository (often master or main)",
	},
	{
		Name:        "created_at",
		Type:        rpc.ColumnTypeDateTime,
		Description: "The date and time the repository was created (RFC3339 format)",
	},
	{
		Name:        "pushed_at",
		Type:        rpc.ColumnTypeDateTime,
		Description: "The date and time the repository was last pushed to (RFC3339 format)",
	},
	{
		Name:        "updated_at",
		Type:        rpc.ColumnTypeDateTime,
		Description: "The date and time the repository was last updated (RFC3339 format)",
	},
	{
		Name:        "html_url",
		Type:        rpc.ColumnTypeString,
		Description: "The URL to the repository on GitHub",
	},
	{
		Name:        "clone_url",
		Type:        rpc.ColumnTypeString,
		Description: "The URL to clone the repository using git",
	},
	{
		Name:        "git_url",
		Type:        rpc.ColumnTypeString,
		Description: "The URL to clone the repository using git",
	},
	{
		Name:        "mirror_url",
		Type:        rpc.ColumnTypeString,
		Description: "The URL to mirror the repository using git",
	},
	{
		Name:        "ssh_url",
		Type:        rpc.ColumnTypeString,
		Description: "The URL to clone the repository using SSH",
	},
	{
		Name:        "language",
		Type:        rpc.ColumnTypeString,
		Description: "The primary language of the repository",
	},
	{
		Name:        "is_fork",
		Type:        rpc.ColumnTypeBool,
		Description: "Whether the repository is a fork",
	},
	{
		Name:        "forks_count",
		Type:        rpc.ColumnTypeInt,
		Description: "The number of forks the repository has",
	},
	{
		Name:        "network_count",
		Type:        rpc.ColumnTypeInt,
		Description: "The number of repositories in the network",
	},
	{
		Name:        "open_issues_count",
		Type:        rpc.ColumnTypeInt,
		Description: "The number of open issues the repository has",
	},
	{
		Name:        "stargazers_count",
		Type:        rpc.ColumnTypeInt,
		Description: "The number of stars the repository has",
	},
	{
		Name:        "subscribers_count",
		Type:        rpc.ColumnTypeInt,
		Description: "The number of subscribers the repository has",
	},
	{
		Name:        "size",
		Type:        rpc.ColumnTypeInt,
		Description: "The size of the repository in kilobytes",
	},
	{
		Name: "allow_rebase_merge",
		Type: rpc.ColumnTypeBool,
	},
	{
		Name: "allow_update_branch",
		Type: rpc.ColumnTypeBool,
	},
	{
		Name: "allow_squash_merge",
		Type: rpc.ColumnTypeBool,
	},
	{
		Name: "allow_merge_commit",
		Type: rpc.ColumnTypeBool,
	},
	{
		Name: "allow_auto_merge",
		Type: rpc.ColumnTypeBool,
	},
	{
		Name: "allow_forking",
		Type: rpc.ColumnTypeBool,
	},
	{
		Name: "web_commit_signoff_required",
		Type: rpc.ColumnTypeBool,
	},
	{
		Name: "delete_branch_on_merge",
		Type: rpc.ColumnTypeBool,
	},
	{
		Name:        "topics",
		Type:        rpc.ColumnTypeString,
		Description: "A JSON array of topics the repository has",
	},
	{
		Name:        "custom_properties",
		Type:        rpc.ColumnTypeString,
		Description: "A JSON object of custom properties",
	},
	{
		Name: "archived",
		Type: rpc.ColumnTypeBool,
	},
	{
		Name: "disabled",
		Type: rpc.ColumnTypeBool,
	},
	{
		Name:        "visibility",
		Type:        rpc.ColumnTypeString,
		Description: "The visibility of the repository. One of public, private, internal",
	},
}

var gistSchema = []rpc.DatabaseSchemaColumn{
	{
		Name:        "id",
		Type:        rpc.ColumnTypeString,
		Description: "The ID of the gist",
	},
	{
		Name:        "gist_url",
		Type:        rpc.ColumnTypeString,
		Description: "The URL to the gist",
	},
	{
		Name:        "by",
		Type:        rpc.ColumnTypeString,
		Description: "The username of the user who created the gist",
	},
	{
		Name:        "user_url",
		Type:        rpc.ColumnTypeString,
		Description: "The URL to the user's profile",
	},
	{
		Name:        "description",
		Type:        rpc.ColumnTypeString,
		Description: "The description of the gist",
	},
	{
		Name:        "comments",
		Type:        rpc.ColumnTypeInt,
		Description: "The number of comments on the gist",
	},
	{
		Name:        "public",
		Type:        rpc.ColumnTypeBool,
		Description: "Whether the gist is public",
	},
	{
		Name:        "created_at",
		Type:        rpc.ColumnTypeDateTime,
		Description: "The date and time the gist was created (RFC3339 format)",
	},
	{
		Name:        "updated_at",
		Type:        rpc.ColumnTypeDateTime,
		Description: "The date and time the gist was last updated (RFC3339 format)",
	},
}
