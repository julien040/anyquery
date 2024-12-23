/*
title = "How often are files modified in a directory? (per month in GB)"
description = "Get the total file size in GB of files modified per month in a directory"

plugins = ["file"]

author = "julien040"

tags = ["file", "modification", "size", "monthly"]

arguments = [
{title="directory_path", display_title = "Directory Path", type="string", description="The path to the directory to analyze", regex=".*"}
]
*/

SELECT
    strftime('%Y-%m', last_modified) as month,
    ROUND(SUM(size) / (1024 * 1024 * 1024), 2) as total_size_gb
FROM
    file_list
WHERE
    directory = @directory_path
GROUP BY
    month
ORDER BY
    month;