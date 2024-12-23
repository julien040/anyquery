/*
title = "List all invites in a server"
description = "Retrieve all invite links of a server"

plugins = ["discord"]

author = "julien040"

tags = ["discord", "invites", "server"]

arguments = [
{title="guild_id", display_title="Server ID", type="string", description="The ID of the server to fetch invites from", regex="^[0-9]+$"}
]
*/

SELECT
    *
FROM
    discord_invites('guild_id')
WHERE
    guild_id = @guild_id;