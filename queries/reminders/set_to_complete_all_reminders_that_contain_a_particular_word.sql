/*
title = "Set reminders containing a word as complete"
description = "Mark as complete all reminders that contain a specific word in their name or body"

plugins = ["reminders"]

author = "julien040"

tags = ["reminders", "update", "completion"]

arguments = [
{title="word", display_title = "Word to search for", type="string", description="The word to search for in the name or body of the reminders", regex="^[a-zA-Z0-9_-]+$"}
]*/

UPDATE 
    reminders_items 
SET 
    completed = 1 
WHERE 
    LOWER(name) LIKE '%' || LOWER(@word) || '%' 
    OR LOWER(body) LIKE '%' || LOWER(@word) || '%';