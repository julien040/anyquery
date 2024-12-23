/*
title = "What is the URL of the active tab?"
description = "Get the URL of the currently active tab in the Brave browser"

plugins = ["brave"]

author = "julien040"

tags = ["brave", "tabs", "url"]

arguments = []
*/

SELECT
    url
FROM
    brave_tabs
WHERE
    active = 1;