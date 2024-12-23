/*
title = "Who is the biggest committer on a GitHub repository?"
description = "Get the user with the most commits on a GitHub repository"

plugins = ["github"]

author = "julien040"

tags = ["github", "commits", "statistics"]

arguments = [
{title="repository", display_title = "Repository name (owner/repo format)", type="string", description="The repository to fetch commits from (owner/repo)", regex="^[a-zA-Z0-9_-]+/[a-zA-Z0-9_-]+$"}
]*/

SELECT
    author,
    author_email,
    count(*) as commits
FROM
    github_commits_from_repository(@repository)
GROUP BY
    author,
    author_email
ORDER BY
    commits DESC
LIMIT
    1;