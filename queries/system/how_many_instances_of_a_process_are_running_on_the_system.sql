/*
title = "How many instances of a process are running on the system?"
description = "Count the number of instances of a specific process running on the system"
plugins = ["system"]
author = "julien040"
tags = ["system", "process", "count"]
arguments = [
    {title="process_name", display_title = "Process Name", type="string", description="The name of the process to count instances for", regex="^[a-zA-Z0-9_-]+$"}
]
*/

SELECT
    count(*) as instance_count
FROM
    system_processes
WHERE
    LOWER(name) = LOWER(@process_name);