
/*
title = "Does this user star this repository on GitHub?"
description = "Check if a specific user has starred a particular repository on GitHub."
plugins = ["github"]
author = "julien040"
tags = ["github", "stars", "repository"]
arguments = [
    {title="user", display_title = "GitHub username", type="string", description="The GitHub username to check", regex="^[a-zA-Z0-9_-]+$"},
    {title="repository", display_title = "Repository name (owner/repo format)", type="string", description="The repository to check (owner/repo)", regex="^[a-zA-Z0-9_-]+/[a-zA-Z0-9_-]+$"}
]
*/

SELECT 
    CASE 
        WHEN COUNT(*) > 0 THEN 'Yes'
        ELSE 'No'
    END as result
FROM 
    github_stars_from_user
WHERE 
    user = @user 
    AND LOWER(full_name) = LOWER(@repository);