/*
title = "How many cores does the system have?"
description = "Get the number of CPU cores in the system"

plugins = ["system"]

author = "julien040"

tags = ["cpu", "system", "cores"]

arguments = []
*/

SELECT cpu_cores FROM system_cpu_infos LIMIT 1;