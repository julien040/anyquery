# Edge plugin

Query and modify tabs of a Chromium based browser.

## Installation

```bash
anyquery install edge
```

## Usage

```sql
-- List all tabs
SELECT * FROM edge_tabs;
-- Close tabs with a specific URL
DELETE FROM edge_tabs WHERE url='https://gut-cli.dev/';
-- Update the url of a tab
UPDATE edge_tabs SET url='https://hn-recommend.julienc.me' WHERE url = 'https://julienc.me';
-- Open a new tab
INSERT INTO edge_tabs (url) VALUES ('https://julienc.me');
```

## Schema

| Column index | Column name | type    |
| ------------ | ----------- | ------- |
| 0            | id          | INTEGER |
| 1            | title       | TEXT    |
| 2            | url         | TEXT    |
| 3            | window_name | TEXT    |
| 4            | window_id   | INTEGER |
| 5            | active      | INTEGER |
| 6            | loading     | INTEGER |

## Caveats

- Update can only be done on the `url` column. Any other column will be ignored.
