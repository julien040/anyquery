/*
title = "Which file extensions are the most common in a directory?"
description = "Get the most common file extensions in a given directory to understand the types of files it contains"

plugins = ["file"]

author = "julien040"

tags = ["file", "extensions", "statistics"]

arguments = [
{title="directory_path", display_title = "Directory Path", type="string", description="The directory path to list files from", regex=".*"}
]
*/

SELECT
    SUBSTR(file_name, INSTR(file_name, '.') + 1) AS extension,
    COUNT(*) AS count
FROM
    file_list(@directory_path)
WHERE
    is_directory = 0
GROUP BY
    extension
ORDER BY
    count DESC
LIMIT 25;