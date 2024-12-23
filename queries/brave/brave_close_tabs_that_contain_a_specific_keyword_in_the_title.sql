/*
title = "Close tabs that contain a specific keyword in the title"
description = "Close all tabs in the Brave browser that have a specific keyword in their title"

plugins = ["brave"]

author = "julien040"

tags = ["brave", "tabs", "close"]

arguments = [
{title="keyword", display_title = "Keyword", type="string", description="Keyword to find in the tab titles", regex=".*"}
]*/

DELETE FROM brave_tabs
WHERE LOWER(title) LIKE LOWER(CONCAT('%', @keyword, '%'));