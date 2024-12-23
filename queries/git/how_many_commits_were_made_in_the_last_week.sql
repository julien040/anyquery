/*
title = "How many commits were made in the last week?"
description = "Count the number of commits made in the last week in a Git repository."

plugins = ["git"]

author = "julien040"

tags = ["git", "commits", "statistics"]

arguments = [
    {title="repository", display_title = "Repository Path", type="string", description="The path to the local or remote Git repository", regex=".*"}
]
*/

SELECT
    COUNT(*) AS commit_count
FROM
    git_commits(@repository)
WHERE
    DATE(author_date) >= DATE('now', '-7 days');