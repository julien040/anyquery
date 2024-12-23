/*
title = "List all tasks that are due today"
description = "Retrieve all tasks from a specific list that have a due date set to today."

plugins = ["clickup"]

author = "julien040"

tags = ["clickup", "tasks", "due date"]

arguments = [
    {title="list_id", display_title="List ID", type="string", description="The ID of the list to fetch tasks from", regex="^[a-zA-Z0-9_-]+$"}
]
*/

SELECT
    *
FROM
    clickup_tasks
WHERE
    list_id = @list_id
    AND date(due_at) = date('now');