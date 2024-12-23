/*
title = "How much memory is used by the system?"
description = "Get the amount of memory currently used by the system"
plugins = ["system"]
author = "julien040"
tags = ["system", "memory", "usage"]
arguments = []
*/

SELECT
    used
FROM
    system_memory_stats;