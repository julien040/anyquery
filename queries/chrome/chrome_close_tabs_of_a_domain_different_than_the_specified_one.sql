/*
title = "Chrome Close tabs of a domain different than the specified one"
description = "Close all tabs from a domain that is different from the specified one"

plugins = ["chrome"]

author = "julien040"

tags = ["chrome", "tabs", "close"]

arguments = [
    {title="domain", display_title = "Domain", type="string", description="The domain to keep tabs open for", regex="^[a-zA-Z0-9.-]+$"}
]*/

DELETE FROM chrome_tabs
WHERE LOWER(url) NOT LIKE CONCAT('%', LOWER(@domain), '%');