/*
title = "List all lists in a specific space"
description = "Retrieve all the lists within a given space in ClickUp."

plugins = ["clickup"]

author = "julien040"

tags = ["clickup", "lists", "space"]

arguments = [
    {title="space_id", display_title = "Space ID", type="string", description="The ID of the space to retrieve lists from", regex="^[0-9]+$"}
]
*/

WITH folder_lists AS (
    SELECT folder_id, name AS folder_name
    FROM clickup_folders
    WHERE space_id = @space_id
)
SELECT
    fl.folder_name,
    l.name AS list_name
FROM
    folder_lists fl
JOIN
    clickup_lists l
ON
    fl.folder_id = l.folder_id;