/*
title = "Which open issues are assigned to me?"
description = "Discover the issues assigned to you"

plugins = ["github"]

author = "julien040"

tags = ["github", "issues", "assigned"]

 */
SELECT
    '#' || "number" as id,
    by,
    created_at,
    title,
    repository,
    'https://github.com' || repository || '/issues/' || "number" as url
FROM
    github_my_issues ('assigned')
WHERE
    state <> 'closed'
    AND is_pull_request = false;