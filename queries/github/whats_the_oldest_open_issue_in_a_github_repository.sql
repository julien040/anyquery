/*
title = "What's the oldest open issue in a GitHub repository?"
description = "Find the oldest open issue in a specific GitHub repository based on the creation date"

plugins = ["github"]

author = "julien040"

tags = ["github", "issues", "oldest"]

arguments = [
    {title="repository", display_title = "Repository name (owner/repo format)", type="string", description="The repository to fetch the oldest open issue from (owner/repo)", regex="^[a-zA-Z0-9_-]+/[a-zA-Z0-9_-]+$"}
]
*/

SELECT 
    id,
    number,
    title,
    body,
    state,
    by,
    assignees,
    labels,
    created_at,
    updated_at,
    url
FROM 
    github_issues_from_repository
WHERE 
    repository = @repository
    AND LOWER(state) = 'open'
ORDER BY 
    datetime(created_at) ASC
LIMIT 1;