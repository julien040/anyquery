/*
title = "List all the contributors of a GitHub repository."
description = "Retrieve a list of all the contributors for a specified GitHub repository."

plugins = ["github"]

author = "julien040"

tags = ["github", "contributors", "repository"]

arguments = [
    {title="repository", display_title="Repository Name (owner/repo format)", type="string", description="The GitHub repository to fetch contributors from (owner/repo)", regex="^[a-zA-Z0-9_-]+/[a-zA-Z0-9_-]+$"}
]
*/

SELECT
    name,
    contributor_url,
    additions,
    deletions,
    commits
FROM
    github_contributors_from_repository
WHERE
    repository = @repository;