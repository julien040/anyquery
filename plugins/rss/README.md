# Rss plugin

Query RSS feeds with SQL.

## Setup

```bash
anyquery install rss
```

## Usage

```sql
-- List all feeds
SELECT * FROM rss_items('http://www.reddit.com/.rss');
-- List all feeds with a filter by the first author
SELECT * FROM rss_items('http://www.reddit.com/.rss') WHERE authors ->> '$[0].name' = 'author_name';
-- Print the first link of every item
SELECT links ->> '$[0]' as link FROM rss_items('http://www.reddit.com/.rss');
```

## Schema

| Column index | Column name | type |
| ------------ | ----------- | ---- |
| 0            | guid        | TEXT |
| 1            | title       | TEXT |
| 2            | description | TEXT |
| 3            | content     | TEXT |
| 4            | links       | TEXT |
| 5            | updated     | TEXT |
| 6            | published   | TEXT |
| 7            | authors     | TEXT |
| 8            | image_url   | TEXT |
| 9            | image_title | TEXT |
| 10           | categories  | TEXT |
| 11           | enclosures  | TEXT |

## Caveats

- The plugin does not do any caching, so it will fetch the feed every time you query it.
- The plugin does not handle authentication (like setting a user agent or cookies).
- The plugin does not handle pagination. You will need to do UNION ALL queries between different pages (.e.g the url has ?page=1, ?page=2, etc).
