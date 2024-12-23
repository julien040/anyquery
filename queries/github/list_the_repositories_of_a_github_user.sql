/*
title = "List the repositories of a GitHub user"
description = "Get a list of repositories from a specific GitHub user"

plugins = ["github"]

author = "julien040"

tags = ["github", "repositories", "user"]

arguments = [
  {title="user", display_title = "GitHub Username", type="string", description="The GitHub username whose repositories need to be listed", regex="^[a-zA-Z0-9_-]+$"}
]
*/

SELECT
    id,
    node_id,
    owner,
    name,
    full_name,
    description,
    homepage,
    default_branch,
    created_at,
    pushed_at,
    updated_at,
    html_url,
    clone_url,
    git_url,
    mirror_url,
    ssh_url,
    language,
    is_fork,
    forks_count,
    network_count,
    open_issues_count,
    stargazers_count,
    subscribers_count,
    size,
    allow_rebase_merge,
    allow_update_branch,
    allow_squash_merge,
    allow_merge_commit,
    allow_auto_merge,
    allow_forking,
    web_commit_signoff_required,
    delete_branch_on_merge,
    topics,
    custom_properties,
    archived,
    disabled,
    visibility
FROM
    github_repositories_from_user
WHERE
    user = @user;