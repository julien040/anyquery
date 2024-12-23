/*
title = "Close tabs of a domain different than the specified one"
description = "Close all tabs that do not belong to the specified domain"

plugins = ["edge"]

author = "julien040"

tags = ["edge", "tabs", "close", "domain"]

arguments = [
{title="domain", display_title = "Domain", type="string", description="The domain to keep tabs open for", regex="^[a-zA-Z0-9.-]+$"}
]
*/

DELETE FROM edge_tabs
WHERE LOWER(url) NOT LIKE CONCAT('%', LOWER(@domain), '%');