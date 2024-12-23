/*
title = "Close tabs with a specific keyword in the title"
description = "Close all tabs that contain a given keyword in their title"

plugins = ["edge"]

author = "julien040"

tags = ["edge", "tabs", "close"]

arguments = [
{title="keyword", display_title = "Keyword", type="string", description="The keyword to search for in the tab titles", regex="^.*$"}
]
*/

DELETE FROM 
    edge_tabs 
WHERE 
    LOWER(title) LIKE LOWER(CONCAT('%', @keyword, '%'));