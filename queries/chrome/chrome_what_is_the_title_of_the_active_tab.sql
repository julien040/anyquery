/*
title = "What is the title of the active tab?"
description = "Get the title of the currently active tab in a Chromium-based browser"

plugins = ["chrome"]

author = "julien040"

tags = ["chrome", "tabs", "active"]

arguments = []
*/

SELECT
    title
FROM
    chrome_tabs
WHERE
    active = 1;