/*
title = "How many messages were sent in a channel?"
description = "Get the total number of messages sent in a specific Discord channel."

plugins = ["discord"]

author = "julien040"

tags = ["discord", "messages", "statistics"]

arguments = [
    {title = "channel_id", display_title = "Channel ID", type = "string", description = "The ID of the channel to count messages from", regex = "^[0-9]+$"}
]
*/

SELECT
    COUNT(*) as total_messages
FROM
    discord_messages
WHERE
    channel_id = @channel_id;