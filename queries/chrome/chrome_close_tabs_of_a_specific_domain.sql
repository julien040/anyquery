/*
title = "Chrome Close tabs of a specific domain"
description = "Close all tabs that belong to a specific domain"

plugins = ["chrome"]

author = "julien040"

tags = ["chrome", "tabs", "management"]

arguments = [
    {title="domain", display_title = "Domain", type="string", description="The domain of the tabs to close", regex="^[a-zA-Z0-9.-]+$"}
]
*/

DELETE FROM chrome_tabs
WHERE url LIKE CONCAT('%', @domain, '%');