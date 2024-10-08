/*
title = "GitHub Stars per day"
description = "Discover the number of stars per day for a given repository ordered by date (recent first)"

plugins = ["github"]

author = "julien040"

tags = ["github", "stars", "statistics"]

arguments = [
{title="repository", display_title = "Repository name (owner/repo format)", type="string", description="The repository to fetch stars from (owner/repo)", regex="^[a-zA-Z0-9_-]+/[a-zA-Z0-9_-]+$"}
]

 */
SELECT
    date (starred_at) AS day,
    count(*) as stars
FROM
    github_stargazers_from_repository(@repository)
GROUP BY
    DAY
ORDER BY
    day DESC;