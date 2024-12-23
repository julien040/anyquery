/*
title = "How many tasks exist per status in a specific list?"
description = "Count the number of tasks per status in a given list"

plugins = ["clickup"]

author = "julien040"

tags = ["clickup", "tasks", "status", "count"]

arguments = [
    {title="list_id", display_title = "List ID", type="string", description="The ID of the list to query for tasks", regex="^[a-zA-Z0-9_-]+$"}
]
*/

SELECT 
    status, 
    COUNT(*) AS task_count 
FROM 
    clickup_tasks 
WHERE 
    list_id = @list_id 
GROUP BY 
    status;