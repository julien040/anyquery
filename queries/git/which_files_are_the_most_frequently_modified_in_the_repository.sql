/*
title = "Which files are the most frequently modified in the repository?"
description = "Find the files that have been modified the most number of times in a given Git repository"

plugins = ["git"]

author = "julien040"

tags = ["git", "files", "modifications"]

arguments = [
    {title="repository_path", display_title = "Repository Path", type="string", description="The path to the Git repository", regex=".*"}
]
*/

SELECT
    file_name,
    COUNT(*) as modifications
FROM
    git_commits_diff
WHERE
    repository_path = @repository_path
GROUP BY
    file_name
ORDER BY
    modifications DESC
LIMIT
    10;