/*
title = "List the open files of a process"
description = "Retrieve the list of files opened by a specific process."

plugins = ["system"]

author = "julien040"

tags = ["system", "process", "files", "open files"]

arguments = [
    {title="pid", display_title = "Process ID", type="int", description="The process ID to fetch the open files for", regex="^\\d+$"}
]
*/

SELECT
    path,
    file_descriptor
FROM
    system_process_files
WHERE
    pid = @pid;