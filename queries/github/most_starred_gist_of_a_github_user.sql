/*
title = "Most starred gist of a GitHub user"
description = "Find the gist with the most stars for a given GitHub user."

plugins = ["github"]

author = "julien040"

tags = ["github", "gists", "stars", "statistics"]

arguments = [
{title="username", display_title = "GitHub Username", type="string", description="The GitHub username to fetch gists from", regex="^[a-zA-Z0-9_-]+$"}
]
*/

SELECT
    id,
    gist_url,
    by,
    description,
    comments,
    created_at,
    updated_at
FROM
    github_gists_from_user
WHERE
    user = @username
ORDER BY
    comments DESC
LIMIT
    1;