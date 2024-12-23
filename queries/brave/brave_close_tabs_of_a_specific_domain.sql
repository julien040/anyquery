/*
title = "Close tabs of a specific domain"
description = "Close all tabs that belong to a specific domain"

plugins = ["brave"]

author = "julien040"

tags = ["brave", "tabs", "close"]

arguments = [
{title = "domain", display_title = "Domain", type = "string", description = "The domain of the tabs to close", regex = "^[a-zA-Z0-9.-]+$"}
]
*/

DELETE FROM brave_tabs
WHERE LOWER(url) LIKE CONCAT('%', LOWER(@domain), '%');