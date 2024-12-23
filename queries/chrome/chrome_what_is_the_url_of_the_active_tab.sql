/*
title = "What is the URL of the active tab?"
description = "Get the URL of the currently active tab in the Chromium based browser"

plugins = ["chrome"]

author = "julien040"

tags = ["chrome", "tabs", "active tab"]
*/

SELECT
    url
FROM
    chrome_tabs
WHERE
    active = 1;