/*
title = "List the child processes of a process"
description = "Get a list of all child processes of a given process ID"

plugins = ["system"]

author = "julien040"

tags = ["system", "processes", "child processes"]

arguments = [
{title="parent_pid", display_title = "Parent Process ID", type="int", description="The process ID of the parent process", regex="^\\d+$"}
]
*/

SELECT *
FROM system_processes
WHERE parent_pid = @parent_pid;