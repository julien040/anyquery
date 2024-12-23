/*
title = "How many containers are running?"
description = "Retrieve the count of currently running containers."
plugins = ["docker"]
author = "julien040"
tags = ["docker", "containers", "status"]
*/

SELECT
    COUNT(*) as running_containers
FROM
    docker_containers
WHERE
    LOWER(state) = 'running';