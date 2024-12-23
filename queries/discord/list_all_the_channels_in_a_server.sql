/*
title = "List all the channels in a server"
description = "Get all channels of a specific Discord server, ordered by their positions."
plugins = ["discord"]
author = "julien040"
tags = ["discord", "channels", "server"]
arguments = [
    {title="guild_id", display_title = "Guild ID", type="string", description="The ID of the server to fetch channels from", regex="^[0-9]+$"}
]
*/

SELECT
    *
FROM
    discord_channels
WHERE
    guild_id = @guild_id
ORDER BY
    position;