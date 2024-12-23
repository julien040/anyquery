/*
title = "Does this user follow this other user on GitHub?"
description = "Check if one user follows another on GitHub"

plugins = ["github"]

author = "julien040"

tags = ["github", "followers", "relationship"]

arguments = [
    {title = "user1", display_title = "User 1", type = "string", description = "The username of the follower", regex = "^[a-zA-Z0-9_-]+$"},
    {title = "user2", display_title = "User 2", type = "string", description = "The username of the followee", regex = "^[a-zA-Z0-9_-]+$"}
]
*/

SELECT 
    CASE 
        WHEN EXISTS (
            SELECT 1 
            FROM github_following_from_user 
            WHERE user = @user1 
              AND follower = @user2
        ) 
        THEN 'Yes' 
        ELSE 'No' 
    END as follows;