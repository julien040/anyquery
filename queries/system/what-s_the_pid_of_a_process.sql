/*
title = "What's the PID of a process?"
description = "Find the Process ID (PID) of a running process by its name"

plugins = ["system"]

author = "julien040"

tags = ["process", "pid", "system"]

arguments = [
    {title="process_name", display_title="Process Name", type="string", description="The name of the process to find the PID for", regex="^[a-zA-Z0-9_-]+$"}
]
*/

SELECT
    pid
FROM
    system_processes
WHERE
    LOWER(name) = LOWER(@process_name);