/*
title = "Close tabs of a domain different than the specified one"
description = "Close all tabs that do not belong to the specified domain"

plugins = ["brave"]

author = "julien040"

tags = ["brave", "tabs", "management"]

arguments = [
{title="domain", display_title = "Domain", type="string", description="The domain to keep tabs open for", regex="^[a-zA-Z0-9.-]+$"}
]
*/

DELETE FROM brave_tabs
WHERE 
    LOWER(url) NOT LIKE CONCAT('%', LOWER(@domain), '%');