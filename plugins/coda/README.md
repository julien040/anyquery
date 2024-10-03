# Coda

Query and INSERT/UPDATE/DELETE data from a coda table using SQL.

## Configuration

First, install the plugin:

```bash
anyquery install coda
```

The plugin will ask you for the following configuration values:

- `token`: Your Coda API token.
- `doc_id`: The document ID of the Coda document you want to query.
- `table_id`: The table ID of the Coda table you want to query.

Below are the steps to find the `token`, `doc_id`, and `table_id`.

### Coda API Token

To use this plugin, you need to generate a Coda API token. To do this, go to your account settings [https://coda.io/account](https://coda.io/account) and click on the `Generate API Token` button under the `API Settings` section (you need to scroll down to see this section).

You can restrict the token to only have access to specific documents by clicking on the `Add a restriction` button and selecting the documents you want to give access to.

### Document ID

You can find the document ID in the URL of the document where your table is located. The document ID is the part of the URL that comes after `https://coda.io/d/prettyName_`
For example, in `https://coda.io/d/Anyquery-test_d4_V9gUn143/Anyquery-test_foo_bar/#table_aaa`, the document ID is `d4_V9gUn143`. You need to leave out the first part of the document ID which is `d/Anyquery-test_` in this case. Often, this part is the title of the document.

### Table ID

To find it, go to [https://coda.io/account](https://coda.io/account) > Labs (scroll to the bottom) > Enable developer mode.  
Then go to the table you want to access and click on the 3 dots on the left > Copy table ID.

## Usage

The table will automatically infer the schema from the Coda table. You can then query the table using SQL.

```sql
-- Select all rows from the table
SELECT * FROM coda_table;

-- Currencies, peoples, links and lookup are returned as their JSON-LD representation
-- Therefore, you might need to use the ->> operator to extract their JSON value
SELECT currencyCol ->> `$.amount` as amount FROM coda_table
SELECT peopleCol ->> `$.email` as email FROM coda_table
SELECT peopleCol ->> `$.name` as name FROM coda_table
SELECT linkCol ->> `$.url` as url FROM coda_table
SELECT lookupCol ->> `$.name` as name FROM coda_table

-- Insert a row into the table
INSERT INTO coda_table (column1, column2) VALUES (value1, value2);

-- Update a row in the table
UPDATE coda_table SET column1 = value1 WHERE column2 = value2;

-- Delete a row from the table
DELETE FROM coda_table WHERE column1 = value1;
```

To have multiple Coda tables, you can create multiple profiles with different `table_id`s. You'll be prompted again for a `token`, `doc_id`, and `table_id` when creating a new profile.

```bash
anyquery profile new default coda my_coda_profile
```

And then query it like this:

```sql
SELECT * FROM my_coda_profile_coda_table;
```

## Schema

The plugin will automatically infer the schema from the Coda table. The first column is the row ID, is named `id`, and is used as a primary key. The rest of the columns are named after the column names in the Coda table, and obviously depend on the data types of the columns in the Coda table.

## Limitations

- The plugin caches data for 5 minutes. If you want to clear the cache, you can run `SELECT clear_plugin_cache('coda');` and then restart anyquery.
- The plugin only supports querying a single table. If you want to query multiple tables, you can create multiple profiles with different `table_id`s.
- You cannot update attachments, and images. You can insert them though by providing the URL of the attachment or image.
- While INSERT/DELETE supports batching (which makes them faster), UPDATE does not support batching and will be slower for large tables.
- The plugin uses batching, this means that an INSERT/DELETE might not be reflected immediately until the buffer is flushed (25 rows of capacity). You can force the buffer to flush by running a SELECT query (e.g. `SELECT * FROM coda_table LIMIT 1`).
