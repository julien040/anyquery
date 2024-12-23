/*
title = "List my GitHub starred repositories for a particular language"
description = "Fetch the list of repositories starred by the authenticated user that are written in a specified programming language."

plugins = ["github"]

author = "julien040"

tags = ["github", "stars", "repositories"]

arguments = [
    {title="language", display_title="Programming Language", type="string", description="The programming language to filter repositories by", regex="^[a-zA-Z]+$"}
]
*/

SELECT
    full_name,
    description,
    html_url,
    language,
    stargazers_count
FROM
    github_my_stars
WHERE
    LOWER(language) = LOWER(@language);