/*
title = "List the top n biggest images"
description = "Get the top n biggest Docker images sorted by their size"

plugins = ["docker"]

author = "julien040"

tags = ["docker", "images", "size"]

arguments = [
    {title="n", display_title = "Number of images", type="int", description="The number of top biggest images to display", regex="^[0-9]+$"}
]
*/

SELECT
    id,
    repo_tags,
    size
FROM
    docker_images
ORDER BY
    size DESC
LIMIT
    @n;