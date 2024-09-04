---
title: "How to run SQL queries on Airtable?"
description: "Learn to run SQL queries on Airtable using Anyquery. This guide covers installation, API key retrieval, configuration, and executing basic SQL commands on Airtable data."
---

# How to Run SQL Queries on Airtable with Anyquery

Anyquery is a powerful SQL query engine that enables you to run SQL queries on various data sources, including Airtable. This tutorial will guide you through the steps to connect Anyquery to Airtable and start running SQL queries.

## Prerequisites

Before starting, ensure you have the following:
- A working [installation of Anyquery](https://anyquery.dev/docs/#installation)
- An Airtable account
- An Airtable base with at least one table

## Step 1: Install the Airtable Plugin

To interact with Airtable from Anyquery, you need to install the Airtable plugin. Open your terminal and run the following command:

```bash
anyquery install airtable
```

Anyquery will prompt you to provide your Airtable API key, base ID, and table name.

## Step 2: Get Your Airtable API Key

To get your Airtable API key, follow these steps:

1. Go to the [Airtable API page](https://airtable.com/account) and generate an API key.
2. Copy the generated API key.

## Step 3: Get Your Airtable Base ID

To find your Airtable base ID, open your base in Airtable and copy the ID from the URL. It is the string after `https://airtable.com/` and before the first `/`. For example, in the URL `https://airtable.com/appWx9fD5JzAB4TIO/tblnTJJsUb8f7QjiM/viwDj8WIME?blocks=hide`, the base ID is `appWx9fD5JzAB4TIO`.

## Step 4: Get Your Airtable Table Name

The Airtable table name is the name of the table you want to query. You can find it in the top left corner of the table in the Airtable interface.

## Step 5: Configure the Plugin

When prompted by Anyquery, enter the API key, base ID, and table name. You will also be asked if you want to enable caching. Enabling caching is recommended for better performance but might result in outdated data.

## Step 6: Run SQL Queries on Airtable

Once configured, you can start running SQL queries on your Airtable data. Here are some examples:

### List All Records in the Table

```sql
SELECT * FROM airtable_table;
```

### Filter Records Based on a Column Value

```sql
SELECT * FROM airtable_table WHERE column_name = 'value';
```

### Insert a New Record

```sql
INSERT INTO airtable_table (column1, column2) VALUES ('value1', 'value2');
```

### Update a Record

```sql
UPDATE airtable_table SET column1 = 'new_value' WHERE id = 'rec123';
```

### Delete a Record

```sql
DELETE FROM airtable_table WHERE id = 'rec123';
```

## Limitations

- The plugin is subject to Airtable's [API rate limits](https://airtable.com/api). Cold runs cap at 5 requests per second, meaning you can read 500 records per second and insert/modify/delete 45 records per second.
- The plugin does not support complex Airtable field types such as linked records, attachments, and formulas.
- The plugin currently does not support `ALTER TABLE` commands. You need to create the Airtable table with the correct columns from the start.

## Conclusion

You've successfully connected Anyquery to Airtable and run SQL queries on your Airtable data. Now you can explore and manipulate your Airtable data using the power of SQL.

For more details and advanced usage, refer to the [Airtable plugin documentation](https://anyquery.dev/integrations/airtable) and the [Anyquery documentation](https://anyquery.dev/docs/usage/).

Happy querying!
