/*
title = "Which tasks have a specific tag?"
description = "Retrieve tasks that have a specific tag in a given list"

plugins = ["clickup"]

author = "julien040"

tags = ["clickup", "tasks", "tags", "filters"]

arguments = [
{title="list_id", display_title = "List ID", type="string", description="The ID of the list to fetch tasks from", regex="^[0-9]+$"},
{title="tag", display_title = "Tag", type="string", description="The tag to filter tasks by", regex="^[a-zA-Z0-9_]+$"}
]
*/

SELECT
    *
FROM
    clickup_tasks
WHERE
    list_id = @list_id
    AND LOWER(tags) LIKE CONCAT('%', LOWER(@tag), '%');