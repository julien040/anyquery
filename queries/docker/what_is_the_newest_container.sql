/*
title = "What is the newest container?"
description = "Find the most recently created container."

plugins = ["docker"]

author = "julien040"

tags = ["docker", "containers", "newest"]

arguments = []
*/

SELECT
    *
FROM
    docker_containers
ORDER BY
    created_at DESC
LIMIT
    1;