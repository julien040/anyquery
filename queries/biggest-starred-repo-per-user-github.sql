/*
title = "What are the 10 most-starred repositories of a user?"
description = "Get the 10 repositories with the most stars of a user"

plugins = ["github"]

author = "julien040"

tags = ["github", "stars", "repositories"]

arguments = [
{title="user", display_title = "GitHub Username", type="string", description="The login username", regex="^[a-zA-Z0-9_-]+$"}
]
 */
SELECT
    full_name as repository,
    stargazers_count as stars,
    forks_count as forks,
    html_url as url
FROM
    github_repositories_from_user (@user)
ORDER BY
    stargazers_count DESC
LIMIT
    10;