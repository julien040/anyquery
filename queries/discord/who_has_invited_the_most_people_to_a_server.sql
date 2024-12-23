/*
title = "Who has invited the most people to a server?"
description = "Find out which user has invited the most people to a specific Discord server."

plugins = ["discord"]

author = "julien040"

tags = ["discord", "invites", "users", "statistics"]

arguments = [
{title="guild_id", display_title = "Server ID", type="string", description="The ID of the server to check invites for", regex="^[0-9]+$"}
]
*/

SELECT 
    created_by_name AS username,
    created_by_id AS user_id,
    SUM(uses) AS total_invites
FROM 
    discord_invites
WHERE 
    guild_id = @guild_id
GROUP BY 
    created_by_name, created_by_id
ORDER BY 
    total_invites DESC
LIMIT 
    1;