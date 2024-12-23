/*
title = "My most starred GitHub repository"
description = "Find the repository with the highest number of stars for the authenticated user."

plugins = ["github"]

author = "julien040"

tags = ["github", "stars", "repositories", "statistics"]

arguments = []*/

SELECT
    full_name,
    stargazers_count
FROM
    github_my_repositories
ORDER BY
    stargazers_count DESC
LIMIT
    1;