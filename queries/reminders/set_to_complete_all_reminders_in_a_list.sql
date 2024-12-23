/*
title = "Set all reminders in a list to complete"
description = "Mark all reminders in a specified list as completed"

plugins = ["reminders"]

author = "julien040"

tags = ["reminders", "update", "completion"]

arguments = [
{title="list_name", display_title = "List Name", type="string", description="The name of the list where reminders will be marked as completed", regex="^[a-zA-Z0-9_ ]+$"}
]*/

UPDATE reminders_items
SET completed = 1
WHERE LOWER(list) = LOWER(@list_name);