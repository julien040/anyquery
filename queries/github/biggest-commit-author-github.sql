/*
title = "What are the 10 biggest commit authors on GitHub?"
description = "Get the 10 users with the most commits on GitHub for a given repository"

plugins = ["github"]

author = "julien040"

tags = ["github", "commits", "statistics"]

arguments = [
{title="repository", display_title = "Repository name (owner/repo format)", type="string", description="The repository to fetch stars from (owner/repo)", regex="^[a-zA-Z0-9_-]+/[a-zA-Z0-9_-]+$"}
]
 */
SELECT
    author,
    author_email,
    count(*) as commits
FROM
    github_commits_from_repository (@repository)
GROUP BY
    author,
    author_email
ORDER BY
    commits DESC
LIMIT
    10;