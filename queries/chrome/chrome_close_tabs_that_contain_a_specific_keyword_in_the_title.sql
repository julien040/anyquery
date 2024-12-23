/*
title = "Close tabs that contain a specific keyword in the title"
description = "Close all tabs in a Chromium-based browser where the title contains a specified keyword"

plugins = ["chrome"]

author = "julien040"

tags = ["chrome", "tabs", "close"]

arguments = [
{title="keyword", display_title = "Keyword", type="string", description="Keyword to search for in the tab titles", regex=".*"}
]
*/

DELETE FROM
    chrome_tabs
WHERE
    LOWER(title) LIKE CONCAT('%', LOWER(@keyword), '%');