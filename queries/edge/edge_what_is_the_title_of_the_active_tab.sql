/*
title = "What is the title of the active tab?"
description = "Retrieve the title of the currently active tab in a Chromium-based browser"

plugins = ["edge"]

author = "julien040"

tags = ["edge", "browser", "tabs"]

arguments = []
*/

SELECT
    title
FROM
    edge_tabs
WHERE
    active = 1
LIMIT 1;