/*
title = "Which process binds to a particular port?"
description = "Find the process that is binding to a specific network port"

plugins = ["system"]

author = "julien040"

tags = ["system", "process", "network"]

arguments = [
{title="port", display_title="Port number", type="int", description="The port number to check which process is binding to it", regex="^[0-9]+$"}
]
 */
SELECT
    p.pid,
    p.name,
    p.cmdline,
    n.local_address,
    n.remote_address,
    n.status
FROM
    system_processes p
    JOIN system_process_networks n ON p.pid = n.pid
WHERE
    json_extract (n.local_address, '$.port') = @port