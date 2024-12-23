/*
title = "Export a ClickUp document to text"
description = "Retrieve and export the content of all pages within a specified ClickUp document as a single concatenated text."

plugins = ["clickup"]

author = "julien040"

tags = ["clickup", "document", "export"]

arguments = [
    {title="workspace_id", display_title = "Workspace ID", type="string", description="The ID of the workspace containing the document to export.", regex="^[0-9]+$"},
    {title="document_id", display_title = "Document ID", type="string", description="The ID of the document to export.", regex="^[A-Z0-9a-z\\-]+$"}
]
*/

SELECT
    GROUP_CONCAT(content, '\n') AS document_text
FROM
    clickup_pages
WHERE
    workspace_id = @workspace_id AND
    doc_id = @document_id;