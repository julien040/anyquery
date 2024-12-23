/*
title = "List all images that don't have a container"
description = "Get all Docker images that are not used by any containers"

plugins = ["docker"]

author = "julien040"

tags = ["docker", "images", "containers", "unused"]

arguments = []
*/

SELECT
    id,
    created_at,
    labels,
    parent_id,
    repo_tags,
    repo_digests,
    container_count,
    shared_size,
    size
FROM
    docker_images
WHERE
    container_count = 0;