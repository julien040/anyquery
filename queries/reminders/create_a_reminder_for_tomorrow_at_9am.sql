/*
title = "Create a reminder for tomorrow at 9am"
description = "Add a new reminder scheduled for tomorrow at 9am in Apple's Reminders app"

plugins = ["reminders"]

author = "julien040"

tags = ["reminders", "create", "todo"]

arguments = [
{title="name", display_title = "Reminder name", type="string", description="The name of the reminder", regex="^.*$"},
{title="body", display_title = "Reminder body", type="string", description="The body/description of the reminder", regex="^.*$"},
{title="list", display_title = "List name", type="string", description="The name of the list to add the reminder to", regex="^.*$"}
]
*/

INSERT INTO reminders_items (name, body, due_date, list) 
VALUES (@name, @body, DATETIME('now', '+1 day', '9 hours'), @list);