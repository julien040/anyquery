/*
title = "How many tabs are open in the browser?"
description = "Get the total number of open tabs in the Chromium-based browser"

plugins = ["edge"]

author = "julien040"

tags = ["edge", "tabs", "statistics"]

arguments = []
*/

SELECT
    COUNT(*) AS open_tabs
FROM
    edge_tabs;