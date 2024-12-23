/*
title = "Who is the last user to join a server?"
description = "Get the most recent user to join a Discord server"

plugins = ["discord"]

author = "julien040"

tags = ["discord", "members", "recent"]

arguments = [
{title="guild_id", display_title="Guild ID", type="string", description="The ID of the Discord server", regex="^[0-9]+$"}
]
*/

SELECT
    user_id,
    username,
    joined_at
FROM
    discord_members
WHERE
    guild_id = @guild_id
ORDER BY
    datetime(joined_at) DESC
LIMIT 1;