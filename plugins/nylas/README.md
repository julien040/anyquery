# Nylas plugin

Query your emails and calendar events using the [Nylas](https://www.nylas.com/) API.

## Setup

```bash
anyquery install nylas
```

## Configuration

Follow the following tutorial to create a sandbox account: [https://developer.nylas.com/docs/v3/quickstart/#create-a-v3-sandbox-application](https://developer.nylas.com/docs/v3/quickstart/#create-a-v3-sandbox-application)

You'll need to save the API key and the grant ID for the plugin configuration. They'll be asked when you install the plugin.

## Usage

```sql
-- Query all events 90 days before today and up to one year in the future
SELECT * FROM nylas_events

-- Query events from a second calendar
SELECT * FROM nylas_events WHERE calendar_id = 'cal_123456'
SELECT * FROM nylas_events('cal_123456')

-- Insert a new event
INSERT INTO nylas_events(title, description,busy, start_at) VALUES ('Meeting with John', 'Discuss of Anyquery', true, '2024-01-01 10:00:00')

-- Update an event
UPDATE nylas_events SET title = 'Meeting with John Doe' WHERE id = 'event_123456'

-- Delete an event
DELETE FROM nylas_events WHERE title = 'Meeting with John Doe'

-- Query all unread emails
SELECT * FROM nylas_emails WHERE unread = true

-- Query all emails from a specific sender
SELECT * FROM nylas_emails WHERE from LIKE '%@example.com'

-- Find an email mentioning a specific keyword
SELECT * FROM nylas_emails WHERE body LIKE '%keyword%'

-- Send an email
INSERT INTO nylas_emails(subject, "to", body) VALUES ('Hello from the Nylas plugin', 'contact@anyquery.dev', 'Hello, this is a test email from the Nylas plugin')

-- Trash an email
DELETE FROM nylas_emails WHERE subject = 'Hello from the Nylas plugin'
```

## Schema

### nylas_events

| Column index | Column name     | type    |
| ------------ | --------------- | ------- |
| 0            | event_id        | TEXT    |
| 1            | title           | TEXT    |
| 2            | description     | TEXT    |
| 3            | created_at      | TEXT    |
| 4            | start_at        | TEXT    |
| 5            | end_at          | TEXT    |
| 6            | location        | TEXT    |
| 7            | status          | TEXT    |
| 8            | busy            | INTEGER |
| 9            | link            | TEXT    |
| 10           | organizer_email | TEXT    |
| 11           | organizer_name  | TEXT    |

### nylas_emails

| Column index | Column name | type    |
| ------------ | ----------- | ------- |
| 0            | id          | TEXT    |
| 1            | subject     | TEXT    |
| 2            | from        | TEXT    |
| 3            | to          | TEXT    |
| 4            | cc          | TEXT    |
| 5            | bcc         | TEXT    |
| 6            | reply_to    | TEXT    |
| 7            | sent_at     | TEXT    |
| 8            | folders     | TEXT    |
| 9            | starred     | INTEGER |
| 10           | unread      | INTEGER |
| 11           | body        | TEXT    |

## Limitations

- The plugin is quite slow (5s per 20 emails). Make sur to use a `LIMIT` clause in your queries.
- The plugin does not do any caching. It will query the Nylas API each time you run a query.

## License

Copyright 2024 Julien CAGNIART

Permission is hereby granted, free of charge, to any person obtaining a copy of this software and associated documentation files (the “Software”), to deal in the Software without restriction, including without limitation the rights to use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of the Software, and to permit persons to whom the Software is furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED “AS IS”, WITHOUT WARRANTY OF ANY KIND, EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

## Additional information

This plugin is submitted for the [Nylas challenge](https://dev.to/challenges/nylas) on [dev.to](https://dev.to/).
