/*
title = "List notes that contain a particular word"
description = "Retrieve all notes that contain a specific word in their content"
plugins = ["notes"]
author = "julien040"
tags = ["notes", "content search"]
arguments = [
  {title="word", display_title = "Word to search for", type="string", description="The word to search for in the notes", regex="^[a-zA-Z0-9]+$"}
]
*/

SELECT 
    * 
FROM 
    notes_items 
WHERE 
    LOWER(html_body) LIKE CONCAT('%', LOWER(@word), '%');
