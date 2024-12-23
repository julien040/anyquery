/*
title = "List all the users in a server"
description = "Get all members of a Discord server"

plugins = ["discord"]

author = "julien040"

tags = ["discord", "members", "server"]

arguments = [
{title="guild_id", display_title = "Server ID", type="string", description="The ID of the Discord server", regex="^[0-9]+$"}
]
*/

SELECT * FROM discord_members WHERE guild_id = @guild_id;