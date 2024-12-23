/*
title = "Which invite has been used the most in a server?"
description = "Find the most used invite code in a Discord server."

plugins = ["discord"]

author = "julien040"

tags = ["discord", "invites", "statistics"]

arguments = [
{title="guild_id", display_title = "Server ID", type="string", description="The ID of the server to fetch invites from", regex="^[0-9]+$"}
]
*/

SELECT
    invide_code,
    MAX(uses) as uses_count
FROM
    discord_invites
WHERE
    guild_id = @guild_id
GROUP BY
    invide_code
ORDER BY
    uses_count DESC
LIMIT
    1;