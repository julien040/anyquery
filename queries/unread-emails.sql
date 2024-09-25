/*
title = "What are my unread emails?"
description = "List all the emails that are unread"

plugins = ["imap"]

author = "julien040"

tags = ["imap", "emails", "unread"]
 */
SELECT
    subject,
    received_at,
    _from as senders
FROM
    imap_emails
WHERE
    flags NOT LIKE '%"Seen"%';