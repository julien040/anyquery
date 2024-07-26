# Pocket plugin

Query, insert and delete articles from [Pocket](https://getpocket.com/).

## Configuration

1. Create a new Pocket app at [https://getpocket.com/developer/apps/new](https://getpocket.com/developer/apps/new).
    ![alt text](https://cdn.jsdelivr.net/gh/julien040/anyquery@main/plugins/pocket/images/registration.png)
2. Copy the consumer key from the app settings.
    ![alt text](https://cdn.jsdelivr.net/gh/julien040/anyquery@main/plugins/pocket/images/copyKey.png)
3. Fill it in the integration server [https://integration.anyquery.dev/pocket](https://integration.anyquery.dev/pocket) and click on `Submit`.
    ![alt text](https://cdn.jsdelivr.net/gh/julien040/anyquery@main/plugins/pocket/images/fillForm.png)
4. Click on `Authorize` to authorize the app.
    ![alt text](https://cdn.jsdelivr.net/gh/julien040/anyquery@main/plugins/pocket/images/authorize.png)
5. Copy the consumer key and access token from the response and fill it in when configuring the plugin.
    ![alt text](https://cdn.jsdelivr.net/gh/julien040/anyquery@main/plugins/pocket/images/success.png)

### Installation

```bash
anyquery install pocket
```

## Usage

### Query

```sql
-- List all articles
SELECT * FROM pocket_items

-- Insert a new article
INSERT INTO pocket_items (given_url, title) VALUES ('https://www.example.com', 'Example article')

-- Delete an article
DELETE FROM pocket_items WHERE given_url = 'https://www.example.com'
```

## Schema

### pocket_items

| Column index | Column name              | type    |
| ------------ | ------------------------ | ------- |
| 0            | id                       | TEXT    |
| 1            | given_url                | TEXT    |
| 2            | given_title              | TEXT    |
| 3            | resolved_url             | TEXT    |
| 4            | resolved_title           | TEXT    |
| 5            | excerpt                  | TEXT    |
| 6            | lang                     | TEXT    |
| 7            | favorite                 | INTEGER |
| 8            | status                   | INTEGER |
| 9            | time_added               | INTEGER |
| 10           | time_updated             | INTEGER |
| 11           | time_favorited           | INTEGER |
| 12           | time_read                | INTEGER |
| 13           | is_article               | INTEGER |
| 14           | has_image                | INTEGER |
| 15           | has_video                | INTEGER |
| 16           | word_count               | INTEGER |
| 17           | time_to_read             | INTEGER |
| 18           | listen_duration_estimate | INTEGER |

## Caveats

- The plugin only supports the `SELECT`, `INSERT` and `DELETE` statements. Updating an article is not yet supported.
- Pocket API has a rate limit of 320 requests per hour. While the plugin automatically caches the data, this solution might not work
if you frequently chain DELETE/INSERT with SELECT statements. This is because the plugin clear the cache after a DELETE/INSERT operation.
- To avoid rate limiting, INSERT/DELETE are buffered (100 operations per batch at the time of writing). This means that an INSERT/DELETE
might not be immediately visible in the SELECT results. To force a push of the buffer, you can run a SELECT statement.
- Requests involving an ORDER BY often involves reading your entire Pocket list. From experience, 2200 articles take around 35 seconds to be read.
