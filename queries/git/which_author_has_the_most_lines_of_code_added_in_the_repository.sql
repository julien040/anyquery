/*
title = "Which author has the most lines of code added in the repository?"
description = "Find the author who has added the most lines of code in the repository"

plugins = ["git"]

author = "julien040"

tags = ["git", "author", "statistics"]

arguments = [
    {title = "repository", display_title = "Repository path or URL", type = "string", description = "The path or URL of the repository to analyze", regex = "^.+$"}
]
*/

SELECT
    author_name,
    SUM(addition) as total_additions
FROM
    git_commits_diff(@repository)
GROUP BY
    author_name
ORDER BY
    total_additions DESC
LIMIT
    1;