# Raindrop

Insert/delete and query your raindrop.io bookmarks with SQL.

## Installation

```bash
anyquery install raindrop
```

## Configuration

1. Go to [https://app.raindrop.io/settings/integrations](https://app.raindrop.io/settings/integrations)
2. Click on "Create new app"
    ![alt text](https://github.com/julien040/anyquery/blob/main/plugins/raindrop/newApp.png)
3. Give it the name you want and click on "Create"
4. Click on the app you just created
5. Click "Create test token" and copy the token
    ![alt text](https://github.com/julien040/anyquery/blob/main/plugins/raindrop/newTestToken.png)
6. Fill it in when requested by `anyquery` in the installation process

## Usage

```sql
-- Insert a bookmark
INSERT INTO raindrop_items(title, link, created_at, reminder) VALUES ('A cool SQL tool', 'https://anyquery.dev', '2024-07-10', '2024-07-20');
-- Delete a bookmark
DELETE FROM raindrop_items WHERE title = 'A cool SQL tool';
-- Query all bookmarks
SELECT * FROM raindrop_items;
```

## Schema

| Column index | Column name     | type    |
| ------------ | --------------- | ------- |
| 0            | id              | INTEGER |
| 1            | link            | TEXT    |
| 2            | title           | TEXT    |
| 3            | excerpt         | TEXT    |
| 4            | note            | TEXT    |
| 5            | user_id         | TEXT    |
| 6            | cover           | TEXT    |
| 7            | tags            | TEXT    |
| 8            | important       | INTEGER |
| 9            | removed         | INTEGER |
| 10           | created_at      | TEXT    |
| 11           | last_updated_at | TEXT    |
| 12           | domain          | TEXT    |
| 13           | collection_id   | INTEGER |
| 14           | reminder        | TEXT    |

## Known limitations

- The plugin does not support the `UPDATE` operation due to rate limiting issues with the Raindrop API. See items.go:377 in the source code for more information.
- The plugin caches your bookmarks for an hour. If you want to clear the cache, you can run `SELECT clear_plugin_cache('raindrop')`, and then restart `anyquery`.
- The plugin buffers the INSERT/DELETE operations and sends them in batches to the Raindrop API. This can lead to a delay in the changes appearing in your Raindrop account. If you want to flush the buffer, run a simple SELECT query like `SELECT * FROM raindrop_items LIMIT 1`.
- When you insert a row, not all fields will be populated in [raindrop.io](https://raindrop.io). For example, id, user_id, removed, domain, etc.
- Tags are represented as a JSON array in the database. You can use the [JSON operator](https://www.sqlite.org/json1.html#the_and_operators) like in PostgreSQL to query the data. For example, `SELECT tags ->> '$[0]' FROM raindrop_items;` will return the first tag. When inserting data, it is expected that you provide a JSON array of tags.
- Date are flexible. While they are always returned as [RFC 3339](https://datatracker.ietf.org/doc/html/rfc3339) strings, you can insert them in the following formats : YYYY-MM-DD, YYYY-MM-DDTHH:MM:SSZ, HH:MM:SS, or a Unix timestamp.
