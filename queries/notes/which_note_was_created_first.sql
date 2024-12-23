
/*
title = "Which note was created first?"
description = "Get the note that was created first."

plugins = ["notes"]

author = "julien040"

tags = ["notes", "creation_date", "first_note"]

arguments = []
*/

SELECT
    id,
    name,
    creation_date,
    modification_date,
    html_body,
    folder,
    account
FROM
    notes_items
ORDER BY
    creation_date ASC
LIMIT 1;