/*
title = "List all subtasks of a specific task"
description = "Retrieve all subtasks of a given task ID."

plugins = ["clickup"]

author = "julien040"

tags = ["clickup", "subtasks", "tasks"]

arguments = [
  {title="task_id", display_title = "Task ID", type="string", description="The ID of the parent task to retrieve subtasks for.", regex="^[a-zA-Z0-9_-]+$"},
  {title="list_id", display_title = "List ID", type="string", description="The ID of the list containing the task.", regex="^[a-zA-Z0-9_-]+$"}
]
*/

SELECT
    *
FROM
    clickup_tasks
WHERE
    parent = @task_id
    AND list_id = @list_id;