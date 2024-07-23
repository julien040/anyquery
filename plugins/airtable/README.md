# Airtable plugin

Use an [Airtable](https://airtable.com/) base as a SQL database.

## Setup

```bash
anyquery install airtable
```

The plugin will request the following information:

1. Airtable API key: Go to the [tokens page](https://airtable.com/create/tokens) and create a new token. Add the scopes:
   - data.records:read
   - data.records:write
   - schema.bases:read
   - schema.bases:write (if you want add new item in a select field with SQL, enable this scope, otherwise you can disable it)
  
    Once done, select the bases you want to query and copy the token.
    Finally, paste the token in the prompt.
2. Airtable base ID. To find it, open your base in Airtable and copy the ID from the URL. It is the string after `https://airtable.com/` and before the first `/`. Example `https://airtable.com/appWx9fD5JzAB4TIO/tblnTJJsUb8f7QjiM/viwDj8WIME?blocks=hide` the base ID is `appWx9fD5JzAB4TIO`.
3. Airtable table name. The name of the table you want to query or its ID. To find the name, open the table in Airtable and copy the name from the top left corner. Example `Table 1`.
4. Enable or disable the cache. If you enable the cache, the plugin will store the data locally for a faster response for an hour. If you disable the cache, the plugin will fetch the data from Airtable every time you run a query. Enabling the cache is recommended for better performance but might result in outdated data. You can delete the cache at any time by running `anyquery -q "SELECT clear_plugin_cache('airtable')"`

## Usage

Each column in the table will be a column in the SQL table. The plugin will automatically infer the data type of each column. References are represented as JSON arrays of record IDs. Users references are represented as JSON arrays of user IDs.

The plugin only adds two columns to the table: `id` and `createdTime`. The `id` column is the record ID in Airtable. The `createdTime` column is the time the record was created.

```sql
-- List all the records in the table
SELECT * FROM airtable_table;
-- List all the records in the table in the view "Grid view"
SELECT * FROM airtable_table('Grid view');
-- Insert a new record
INSERT INTO airtable_table (column1, column2) VALUES ('value1', 'value2');
-- Update a record
UPDATE airtable_table SET column1 = 'new_value' WHERE id = 'rec123';
-- Delete a record
DELETE FROM airtable_table WHERE id = 'rec123';
```

## Limitations

- Due to rate limits, cold runs caps at 5 requests per second. It therefore means you can read 500 records per second, and insert/modify/delete 45 records per second.

## Troubleshooting

- `failed to get records(422): {"error":{"type":"LIST_RECORDS_ITERATOR_NOT_AVAILABLE"}}`: This error occurs when the plugin tries to fetch records from Airtable after a long time they were put in the cache. To fix this, run `anyquery -q "SELECT clear_plugin_cache('airtable')"`, and restart anyquery.
