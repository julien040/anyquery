/*
title = "How many notes are contained in a folder?"
description = "Get the count of notes in a specific folder"

plugins = ["notes"]

author = "julien040"

tags = ["notes", "folder", "count"]

arguments = [
{title="folder_name", display_title = "Folder Name", type="string", description="The name of the folder to count notes in", regex=".*"}
]
*/

SELECT
    COUNT(*) AS notes_count
FROM
    notes_items
WHERE
    LOWER(folder) = LOWER(@folder_name);