/*
title = "Who are the stargazers of a GitHub repository?"
description = "List the users who starred a GitHub repository"

plugins = ["github"]

author = "julien040"

tags = ["github", "stars"]

arguments = [
{title="repository", display_title = "Repository name (owner/repo format)", type="string", description="The repository to fetch stars from (owner/repo)", regex="^[a-zA-Z0-9_-]+/[a-zA-Z0-9_-]+$"}
]
 */
SELECT
    *
FROM
    github_stargazers_from_repository (@repository);