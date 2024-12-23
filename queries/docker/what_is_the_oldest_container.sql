/*
title = "What is the oldest container?"
description = "Find the oldest Docker container based on the created date."
plugins = ["docker"]
author = "julien040"
tags = ["docker", "containers", "oldest"]

arguments = []
*/

SELECT
    id,
    names,
    image,
    created_at
FROM
    docker_containers
ORDER BY
    datetime(created_at) ASC
LIMIT
    1;