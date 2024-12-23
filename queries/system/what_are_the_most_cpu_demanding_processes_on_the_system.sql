/*
title = "What are the most CPU demanding processes on the system?"
description = "Get the processes that are using the most CPU on the system"

plugins = ["system"]

author = "julien040"

tags = ["system", "cpu", "processes", "statistics"]

arguments = [
    {title="limit", display_title = "Limit", type="int", description="The number of processes to return", regex="^[0-9]+$"}
]
*/

SELECT
    p.pid,
    p.name,
    s.cpu_percent
FROM
    system_processes p
    JOIN system_process_stats s ON p.pid = s.pid
ORDER BY
    s.cpu_percent DESC
LIMIT
    @limit;