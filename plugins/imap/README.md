# Imap plugin

Run SQL queries on your mailboxes.

## Setup

```bash
anyquery install imap
```

## Usage

Fill in the required configuration when installing the plugin. When you're done, you can run these SQL queries for example:

```sql
-- List all folders in the mailbox
SELECT * FROM imap_folders;

-- List all emails in the inbox
SELECT * FROM imap_emails;

-- List all emails in the inbox with a specific subject
SELECT * FROM imap_emails WHERE subject LIKE '%important%';

-- List unseen emails in the inbox
SELECT * FROM imap_emails WHERE flags NOT LIKE '%"Seen"%';

-- List flagged emails in the inbox
SELECT * FROM imap_emails WHERE flags LIKE '%"Flagged"%';

-- List all emails in the inbox with a specific sender
SELECT * FROM imap_emails EXISTS (SELECT 1 FROM json_tree(_from) WHERE key = 'email' AND value = '<the sender email');

-- List all emails containing a specific word in the body
SELECT * FROM imap_emails_body WHERE body LIKE '%meeting with John%';
-- 
```

## Schema

### imap_folders

List your folders in your mailbox.

| Column index | Column name | type |
| ------------ | ----------- | ---- |
| 0            | folder      | TEXT |

### imap_emails

List your emails without the body (more performant).

Known flags are `Seen`, `Flagged`, `Answered`, `Deleted`, `Draft`.

| Column index | Column name | type    |
| ------------ | ----------- | ------- |
| 0            | uid         | INTEGER |
| 1            | subject     | TEXT    |
| 2            | sent_at     | TEXT    |
| 3            | received_at | TEXT    |
| 4            | _from       | TEXT    |
| 5            | to          | TEXT    |
| 6            | reply_to    | TEXT    |
| 7            | cc          | TEXT    |
| 8            | bcc         | TEXT    |
| 9            | message_id  | TEXT    |
| 10           | flags       | TEXT    |
| 11           | size        | INTEGER |
| 12           | folder      | TEXT    |

### imap_emails_body

List your emails with the body (less performant, about 70 emails per second on a gigabit connection). Note that it only supports email bodies encoded in UTF-8. Other encodings will be ignored and filled with `NULL`.

| Column index | Column name | type    |
| ------------ | ----------- | ------- |
| 0            | uid         | INTEGER |
| 1            | subject     | TEXT    |
| 2            | sent_at     | TEXT    |
| 3            | received_at | TEXT    |
| 4            | _from       | TEXT    |
| 5            | to          | TEXT    |
| 6            | reply_to    | TEXT    |
| 7            | cc          | TEXT    |
| 8            | bcc         | TEXT    |
| 9            | message_id  | TEXT    |
| 10           | flags       | TEXT    |
| 11           | size        | INTEGER |
| 12           | folder      | TEXT    |
| 13           | body        | TEXT    |

## Caveats and known information

- The plugin caches the emails for 30 days. If you want to refresh the cache, you can run `SELECT clear_plugin_cache('imap');`
- On a Macbook Air M1 with a gigabit connection, the plugin fetches 110 emails per second on average with Outlook and 180 per second with Gmail.
- The plugin is not able to fetch the attachments of the email.
