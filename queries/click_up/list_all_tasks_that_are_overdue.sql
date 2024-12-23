/*
title = "List all tasks that are overdue"
description = "Retrieve all tasks that are overdue in a given list"
plugins = ["clickup"]
author = "julien040"
tags = ["clickup", "tasks", "overdue"]
arguments = [
    {title="list_id", display_title = "List ID", type="string", description="The ID of the list to fetch tasks from", regex="^[0-9]+$"}
]
*/

SELECT
    *
FROM
    clickup_tasks
WHERE
    list_id = @list_id
    AND due_at < datetime('now')
    AND (status IS NULL OR LOWER(status) != 'closed');