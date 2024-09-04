---
title: "How to filter an HTML table using SQL?"
description: "Learn to filter HTML tables using Anyquery's SQL query engine. This tutorial covers prerequisites, querying steps, and exporting results to JSON for easy data handling."
---

# How to Filter an HTML Table Using SQL with Anyquery

Anyquery is a powerful SQL query engine that allows you to run SQL queries on various data sources, including HTML tables. This tutorial will guide you on how to filter an HTML table using SQL.

## Prerequisites

Before starting, ensure you have the following:

- A working installation of Anyquery. For installation instructions, refer to [Anyquery installation documentation](https://anyquery.dev/docs/#installation).

## Step 1: Identify the HTML Table

First, you need to identify the HTML table you want to query. For example, consider the HTML table from [diskprices.com](https://diskprices.com). The table contains information about disk prices.

## Step 2: Query the HTML Table using Anyquery

Use the `read_html` table function in Anyquery to read the HTML table. The `read_html` function takes two arguments: the URL of the HTML page and a CSS selector indicating the table.

Here is an example SQL query to select all rows from the HTML table at `diskprices.com`:

```sql
SELECT * FROM read_html('https://diskprices.com', 'table');
```

## Step 3: Filter the HTML Table

You can filter the HTML table by adding a `WHERE` clause to your SQL query. For example, to filter the table to find the cheapest 12TB HDD, you can run the following query:

```sql
SELECT * 
FROM read_html('https://diskprices.com', 'table') 
WHERE Technology = 'HDD' AND Capacity = '12 TB' 
ORDER BY Price 
LIMIT 1;
```

This SQL query will filter the HTML table to show only the rows where the `Technology` column is 'HDD' and the `Capacity` column is '12 TB'. It then orders the results by the `Price` column and selects the cheapest option with the `LIMIT 1` clause.

## Example: Filter a Table and Export to JSON

If you want to export the filtered data to a JSON file, you can use the `--json` flag:

```bash
anyquery -q "SELECT * FROM read_html('https://diskprices.com', 'table') WHERE Technology = 'HDD' AND Capacity = '12 TB' ORDER BY Price LIMIT 1" --json > cheapest_12TB_HDD.json
```

## Conclusion

You have successfully learned how to filter an HTML table using SQL with Anyquery. Now you can explore and query any HTML table using SQL. For more information, refer to the [official documentation](https://anyquery.dev/docs/usage/querying-files).
