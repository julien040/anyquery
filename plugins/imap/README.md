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

-- List all emails in the inbox with a specific sender
SELECT * FROM imap_emails EXISTS (SELECT 1 FROM json_tree(_from) WHERE key = 'email' AND value = '<the sender email');

```

```bash
# Export all emails as HTML in a JSON file
anyquery -q "SELECT html FROM imap_emails" --json > emails.json
```

## Schema

### imap_folders

| Column index | Column name | type |
| ------------ | ----------- | ---- |
| 0            | folder      | TEXT |

### imap_emails

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

## Caveats and known information

- The plugin caches the emails for 30 days. If you want to refresh the cache, you can run `SELECT clear_plugin_cache('imap');`
- On a Macbook Air M1 with a gigabit connection, the plugin fetches 110 emails per second on average with Outlook and 180 per second with Gmail.
- The plugin is not able to fetch the body of the email, only the "metadata" (subject, sender, etc.).
- The plugin is not able to fetch the attachments of the email.
