/*
title = "List all documents in a specific folder"
description = "Get all documents within a specific folder by providing the folder ID"

plugins = ["clickup"]

author = "julien040"

tags = ["clickup", "documents", "folder"]

arguments = [
{title="folder_id", display_title = "Folder ID", type="string", description="The ID of the folder to fetch documents from", regex="^[0-9]+$"}
]
*/

SELECT
    d.doc_id,
    d.created_at,
    d.updated_at,
    d.name,
    d.parent_id,
    d.creator_id,
    d.deleted
FROM
    clickup_docs d,
    clickup_folders f
WHERE
    f.folder_id = @folder_id
AND
    d.parent_id = f.folder_id;