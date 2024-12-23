/*
title = "How many people were banned in a server?"
description = "Count the number of users who have been banned from a specific Discord server."

plugins = ["discord"]

author = "julien040"

tags = ["discord", "bans", "server"]

arguments = [
    {title="guild_id", display_title = "Server ID", type="string", description="The ID of the Discord server", regex="^[0-9]+$"}
]
*/

SELECT COUNT(*) as banned_users
FROM discord_bans
WHERE id = @guild_id;