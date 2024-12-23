/*
title = "How many messages contain a particular word?"
description = "Count the number of messages in a Discord channel that contain a specific word."

plugins = ["discord"]

author = "julien040"

tags = ["discord", "messages", "count"]

arguments = [
    {title="channel_id", display_title = "Channel ID", type="string", description="The ID of the channel to search messages in.", regex="^[0-9]+$"},
    {title="word", display_title = "Word", type="string", description="The word to search for in messages.", regex="^\\S+$"}
]
*/

SELECT COUNT(*) 
FROM discord_messages 
WHERE channel_id = @channel_id 
AND LOWER(content) LIKE LOWER(CONCAT('%', @word, '%'));