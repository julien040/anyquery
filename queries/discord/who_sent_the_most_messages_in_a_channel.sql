/*
title = "Who sent the most messages in a channel?"
description = "Find the user who sent the most messages in a specific Discord channel."

plugins = ["discord"]

author = "julien040"

tags = ["discord", "messages", "statistics"]

arguments = [
{title="channel_id", display_title = "Channel ID", type="string", description="The ID of the channel to fetch messages from", regex="^[0-9]+$"}
]
*/

SELECT
    username,
    COUNT(*) as message_count
FROM
    discord_messages
WHERE
    channel_id = @channel_id
GROUP BY
    username
ORDER BY
    message_count DESC
LIMIT
    1;