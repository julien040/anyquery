/*
title = "What's the first message sent in a channel?"
description = "Retrieve the first message that was sent in a specified Discord channel."

plugins = ["discord"]

author = "julien040"

tags = ["discord", "messages", "channel"]

arguments = [
    {title="channel_id", display_title = "Channel ID", type="string", description="The ID of the channel to fetch the first message from", regex="^[0-9]+$"}
]
*/

SELECT 
    * 
FROM 
    discord_messages 
WHERE 
    channel_id = @channel_id
ORDER BY 
    datetime(created_at) ASC 
LIMIT 1;