/*
title = "List all containers running a specific image"
description = "Get a list of all Docker containers that are currently running a specified image."

plugins = ["docker"]

author = "julien040"

tags = ["docker", "containers", "images"]

arguments = [
  {title="image_name", display_title = "Image Name", type="string", description="The name of the image to filter containers by.", regex="^[/a-zA-Z0-9._-]+(:[a-zA-Z0-9._-]+)?$"}
]
*/

SELECT
    id,
    names,
    image,
    image_id,
    command,
    created_at,
    ports,
    labels,
    size_rw,
    size_root_fs,
    state,
    status,
    networks,
    mounts
FROM
    docker_containers
WHERE
    LOWER(image) = LOWER(@image_name);