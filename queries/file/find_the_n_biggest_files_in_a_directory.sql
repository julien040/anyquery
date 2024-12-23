/*
title = "Find the n biggest files in a directory"
description = "Get the n largest files by size in a specified directory"

plugins = ["file"]

author = "julien040"

tags = ["files", "size", "directory", "biggest"]

arguments = [
{title="directory", display_title = "Directory Path", type="string", description="The path of the directory to search for files", regex=".*"},
{title="limit", display_title = "Number of files to return", type="int", description="The number of largest files to return", regex="^[0-9]+$"}
]
*/

SELECT
    file_name,
    size,
    path
FROM
    file_list(@directory)
WHERE
    is_directory = 0
ORDER BY
    size DESC
LIMIT
    @limit;