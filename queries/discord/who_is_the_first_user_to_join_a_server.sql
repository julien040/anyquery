/*
title = "Who is the first user to join a server?"
description = "Find the first user to join a specified Discord server."

plugins = ["discord"]

author = "julien040"

tags = ["discord", "members", "joined"]

arguments = [
    {title="guild_id", display_title="Guild ID", type="string", description="The ID of the server to check", regex="^[0-9]+$"}
]
*/

SELECT
    username,
    user_id,
    joined_at
FROM
    discord_members
WHERE
    guild_id = @guild_id
ORDER BY
    joined_at ASC
LIMIT 1;