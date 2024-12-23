/*
title = "List all tasks in a specific list"
description = "Retrieve all tasks from a specified ClickUp list"

plugins = ["clickup"]

author = "julien040"

tags = ["clickup", "tasks", "list"]

arguments = [
{title="list_id", display_title = "List ID", type="string", description="The ID of the list to fetch tasks from", regex="^[0-9]+$"}
]
*/

SELECT
    *
FROM
    clickup_tasks
WHERE
    list_id = @list_id;