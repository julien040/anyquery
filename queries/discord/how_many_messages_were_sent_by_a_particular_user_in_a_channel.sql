
/*
title = "How many messages were sent by a particular user in a channel?"
description = "Count the number of messages sent by a specific user in a particular channel on Discord."

plugins = ["discord"]

author = "julien040"

tags = ["discord", "messages", "user", "count"]

arguments = [
  {title="channel_id", display_title = "Channel ID", type="string", description="The ID of the Discord channel", regex="^[0-9]+$"},
  {title="user_id", display_title = "User ID", type="string", description="The ID of the user", regex="^[0-9]+$"}
]
*/

SELECT
  COUNT(*) AS message_count
FROM
  discord_messages
WHERE
  channel_id = @channel_id
  AND user_id = @user_id;