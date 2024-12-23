/*
title = "What's the last message sent in a channel?"
description = "Retrieve the most recent message sent in a specific Discord channel."

plugins = ["discord"]

author = "julien040"

tags = ["discord", "messages", "recent"]

arguments = [
    {title="channel_id", display_title="Channel ID", type="string", description="The ID of the channel to fetch the last message from", regex="^[0-9]+$"}
]
*/

SELECT
    *
FROM
    discord_messages
WHERE
    channel_id = @channel_id
ORDER BY
    created_at DESC
LIMIT
    1;