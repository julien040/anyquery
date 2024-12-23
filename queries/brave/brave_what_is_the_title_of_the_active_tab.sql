/*
title = "What is the title of the active tab?"
description = "Retrieve the title of the currently active tab in the Brave browser"

plugins = ["brave"]

author = "julien040"

tags = ["brave", "tabs", "active"]

arguments = []
*/

SELECT
    title
FROM
    brave_tabs
WHERE
    active = 1;