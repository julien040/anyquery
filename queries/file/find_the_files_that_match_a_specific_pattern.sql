/*
title = "Find files matching a specific pattern"
description = "Retrieve a list of files that match a specific pattern in their names"

plugins = ["file"]

author = "julien040"

tags = ["file", "search", "pattern"]

arguments = [
    {title="pattern", display_title = "File name pattern", type="string", description="The pattern to search for in file names", regex=".*"}
]
*/

SELECT
    path,
    file_name,
    file_type,
    size,
    last_modified,
    is_directory
FROM
    file_search(@pattern);