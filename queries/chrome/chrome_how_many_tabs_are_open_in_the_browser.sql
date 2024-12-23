/*
title = "How many tabs are open in the browser?"
description = "Get the total number of open tabs in the Chromium-based browser"

plugins = ["chrome"]

author = "julien040"

tags = ["chrome", "tabs", "browser"]
*/

SELECT 
    COUNT(*) AS open_tabs
FROM 
    chrome_tabs;