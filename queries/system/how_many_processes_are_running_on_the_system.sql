/*
title = "How many processes are running on the system?"
description = "Get the total number of processes currently running on the system."

plugins = ["system"]

author = "julien040"

tags = ["system", "process", "count"]

arguments = []
*/

SELECT
    count(*) as total_processes
FROM
    system_processes;