/*
title = "How many downloads got each release of a GitHub repository?"
description = "Get the number of downloads for each release of a GitHub repository"

plugins = ["github"]

author = "julien040"

tags = ["github", "downloads", "releases"]

arguments = [
{title="repository", display_title = "Repository name (owner/repo format)", type="string", description="The repository to fetch stars from (owner/repo)", regex="^[a-zA-Z0-9_-]+/[a-zA-Z0-9_-]+$"}
]

 */
WITH
    releases_assets AS (
        SELECT
            r.name as release_id,
            js.value ->> 'download_count' as downloads
        FROM
            github_releases_from_repository (@repository) r,
            json_each (assets) js
    )
SELECT
    release_id as release,
    sum(downloads) as downloads
FROM
    releases_assets
GROUP BY
    release_id
ORDER BY
    release_id ASC;