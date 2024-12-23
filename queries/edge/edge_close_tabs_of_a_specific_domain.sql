/*
title = "Edge Close tabs of a specific domain"
description = "Close all tabs from a specific domain in a Chromium-based browser."

plugins = ["edge"]

author = "julien040"

tags = ["edge", "browser", "tabs", "close"]

arguments = [
{title="domain", display_title = "Domain", type="string", description="The domain whose tabs should be closed", regex="^[a-zA-Z0-9.-]+$"}
]
*/

DELETE FROM edge_tabs
WHERE url LIKE CONCAT('%', @domain, '%');