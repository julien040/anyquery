/*
title = "List all tasks due in the next n days"
description = "Retrieve all tasks from a specific list that are due within the next n days"
plugins = ["clickup"]
author = "julien040"
tags = ["clickup", "tasks", "due", "date"]
arguments = [
    {title="list_id", display_title = "List ID", type="string", description="The ID of the list to fetch tasks from", regex="^[a-zA-Z0-9_-]+$"},
    {title="days", display_title = "Number of days", type="int", description="The number of days from today to check for due tasks", regex="^[0-9]+$"}
]
*/

SELECT
    task_id,
    description,
    status,
    due_at
FROM
    clickup_tasks
WHERE
    list_id = @list_id
    AND due_at IS NOT NULL
    AND DATE(due_at) <= DATE('now', CONCAT('+', @days, ' days'));