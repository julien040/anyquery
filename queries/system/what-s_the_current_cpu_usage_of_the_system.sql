/*
title = "What's the current CPU usage of the system?"
description = "Get the current CPU usage statistics of the system, including user, system, and idle times."

plugins = ["system"]

author = "julien040"

tags = ["cpu", "usage", "statistics"]
*/

SELECT
    *
FROM
    system_cpu_stats;