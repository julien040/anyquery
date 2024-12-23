/*
title = "How many tabs are open in the browser?"
description = "Count the total number of open tabs in the browser"

plugins = ["brave"]

author = "julien040"

tags = ["brave", "tabs", "count", "open"]

arguments = []
*/

SELECT
    COUNT(*) AS open_tabs
FROM
    brave_tabs;