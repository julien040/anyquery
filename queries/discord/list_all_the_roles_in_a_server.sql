/*
title = "List all the roles in a server."
description = "Retrieve all roles from a specific Discord server."

plugins = ["discord"]

author = "julien040"

tags = ["discord", "roles", "server"]

arguments = [
{title="guild_id", display_title = "Guild ID", type="string", description="The ID of the guild (server) to fetch roles from", regex="^[0-9]+$"}
]*/

SELECT * 
FROM discord_roles
WHERE guild_id = @guild_id;