/*
title = "List all tasks assigned to a specific user"
description = "Retrieve all tasks that are assigned to a specific user using their user ID."

plugins = ["clickup"]

author = "julien040"

tags = ["clickup", "tasks", "user"]

arguments = [
{title="list_id", display_title = "List ID", type="string", description="The ID of the list to fetch tasks from", regex="^[0-9]+$"},
{title="user_id", display_title = "User ID", type="string", description="The ID of the user to filter tasks by", regex="^[0-9]+$"}
]
*/

SELECT
    *
FROM
    clickup_tasks
WHERE
    list_id = @list_id
    AND assignees LIKE CONCAT('%', @user_id, '%');