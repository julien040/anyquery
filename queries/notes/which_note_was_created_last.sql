/*
title = "Which note was created last?"
description = "Find the most recently created note"
plugins = ["notes"]
author = "julien040"
tags = ["notes", "creation_date", "recent"]
*/

SELECT
    *
FROM
    notes_items
ORDER BY
    creation_date DESC
LIMIT 1;