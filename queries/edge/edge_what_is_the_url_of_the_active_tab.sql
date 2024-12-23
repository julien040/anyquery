/*
title = "What is the URL of the active tab?"
description = "Get the URL of the currently active tab"

plugins = ["edge"]

author = "julien040"

tags = ["edge", "tabs"]

arguments = []
*/

SELECT
    url
FROM
    edge_tabs
WHERE
    active = 1;