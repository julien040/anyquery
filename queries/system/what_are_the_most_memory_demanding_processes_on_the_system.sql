/*
title = "What are the most memory demanding processes on the system?"
description = "List the processes that consume the most memory on the system"
plugins = ["system"]
author = "julien040"
tags = ["system", "memory", "processes", "statistics"]
arguments = []
*/

SELECT
    p.pid,
    p.name,
    m.resident_set_size,
    m.virtual_memory_size
FROM
    system_processes p
    JOIN system_process_memory m ON p.pid = m.pid
ORDER BY
    m.resident_set_size DESC
LIMIT
    10;