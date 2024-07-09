# Notion plugin

This plugin allows you to interact with Notion databases. You can read/insert/update/delete records in a database.

## Installation

You need [Anyquery](https://github.com/julien040/anyquery) to run this plugin.

Then, install the plugin with the following command:

```bash
anyquery install notion
```

At some point, you will be asked to provide your Notion API key. You can find it by creating an integration.

### Find your Notion API key

1. Go to [Notion's My Integrations page](https://www.notion.so/my-integrations).
2. Click on the `+ New integration` button.

    ![Home of Notion integrations](https://github.com/julien040/anyquery/blob/main/plugins/notion/images/creator-profile.png)
3. Fill in the form with the following information:
   1. Name: Whatever you want
   2. Associated workspace: The workspace you want the plugin to have access to
   3. Type: Internal

   ![A form to create a new integration](https://github.com/julien040/anyquery/blob/main/plugins/notion/images/form-integration.png)
4. Click on the `Save` button and on `Configure integration settings`.

    ![alt text](https://github.com/julien040/anyquery/blob/main/plugins/notion/images/success.png)
5. Copy the `token` and paste it when asked by the plugin.

    ![alt text](https://github.com/julien040/anyquery/blob/main/plugins/notion/images/token.png)

### Finding the database ID

Once you have your API key, you need to find the database ID of the database you want to interact with. You can find it in the URL of the database. For example, if the URL of the database is `https://www.notion.so/myworkspace/My-Database-1234567890abcdef1234567890abcdef`, the database ID is `1234567890abcdef1234567890abcdef`.

## Usage

The plugin supports all the basic SQL operations. Here are some examples:

```sql
SELECT * FROM notion_database;

SELECT * FROM notion_database WHERE name = 'Michael';

INSERT INTO notion_database (name, age) VALUES ('Michael', 25);

UPDATE notion_database SET age = 26 WHERE name = 'Michael';

DELETE FROM notion_database WHERE name = 'Michael';
```

## Known limitations

- Rollup and UniqueID properties are not supported.
- Due to the nature of formulas, a column can have different types depending on the row. This can lead to unexpected results when filtering records.
- Because SQLite does not support arrays, the plugin will return a JSON representation of the array. For example, `["a", "b", "c"]` will be returned as `'["a", "b", "c"]'`. <br>
You can then use the [JSON operator](https://www.sqlite.org/json1.html#the_and_operators) like in PostgreSQL to query the data. For example, `SELECT "Files & media" ->> '$[0]' FROM notion_database;` will return the first element of the array.
- You cannot create/update files, formulas, or rollup properties. You cannot update the cover and icon properties of a page.
- `DELETE FROM` operations only trash the record. You can restore it from the Notion interface.
- Because SQLite does not have a `BOOLEAN` type, the plugin will return `0` for `false` and `1` for `true`.
- Dates are returned as strings in the format `YYYY-MM-DDTHH:MM:SSZ`(RFC3339). If an end date is specified, it will be returned as a string in the format `YYYY-MM-DDTHH:MM:SSZ/YYYY-MM-DDTHH:MM:SSZ`. <br>
When inserting/updating a date, you can specify the date as YYYY-MM-DD, DD/MM/YYYY, RFC3339, or a Unix timestamp. If you want to specify a time, you can use the format `YYYY-MM-DDTHH:MM:SSZ`.
- Rate limit: Notion has a rate limit of 3 requests per second. While the plugin automatically handles retries, it may slow down the execution of your queries.
For example, if you run a query that inserts 100 records, it will take at least 33 seconds to complete. And you will read at most 300 records per second.
