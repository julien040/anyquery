/*
title = "List all the branches of a GitHub repository"
description = "Get a list of all branches for a specific GitHub repository"

plugins = ["github"]

author = "julien040"

tags = ["github", "branches", "repository"]

arguments = [
    {title="repository", display_title = "Repository name (owner/repo format)", type="string", description="The repository to fetch branches from (owner/repo)", regex="^[a-zA-Z0-9_-]+/[a-zA-Z0-9_-]+$"}
]
*/

SELECT
    name,
    commit_sha,
    protected,
    url
FROM
    github_branches_from_repository
WHERE
    repository = @repository;