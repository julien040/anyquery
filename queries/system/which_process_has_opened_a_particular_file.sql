/*
title = "Which process has opened a particular file?"
description = "Find the process that has opened a specific file"

plugins = ["system"]

author = "julien040"

tags = ["system", "process", "files"]

arguments = [
    {title="file_path", display_title="File Path", type="string", description="The full path of the file to search for", regex="^.*$"}
]
*/

SELECT
    p.pid,
    p.name,
    p.exe,
    p.cmdline
FROM
    system_processes p
    JOIN system_process_files f ON p.pid = f.pid
WHERE
    LOWER(f.path) = LOWER(@file_path);