---
title: "How to filter a CSV file by a column value?"
description: "Learn how to filter a CSV file by a specific column value using Anyquery. Follow step-by-step instructions to query, filter, and optionally export your data."
---

# How to Filter a CSV File by a Column Value

Anyquery is a SQL query engine that allows you to run SQL queries on various data sources, including CSV files. This tutorial will guide you through the steps to filter a CSV file by a specific column value using Anyquery.

## Prerequisites

Before starting, ensure you have the following:

- A working installation of Anyquery. For installation instructions, refer to the [Anyquery documentation](https://anyquery.dev/docs/#installation).

## Step 1: Prepare Your CSV File

Ensure your CSV file is properly formatted and accessible. For the sake of this tutorial, let's assume you have a CSV file named `example.csv` with the following content:

```csv
id,name,age,city
1,John,30,New York
2,Jane,25,Los Angeles
3,Bob,35,Chicago
4,Alice,28,San Francisco
5,Eve,22,New York
```

## Step 2: Launch the Anyquery Shell

Open your terminal and start the Anyquery shell:

```bash
anyquery
```

## Step 3: Query the CSV File

Use the `read_csv` function to query the CSV file. The `read_csv` table function requires the path to your CSV file. To filter rows by a specific column value, you can use a `WHERE` clause in your SQL query.

### Example Query: Filter by City "New York"

To filter rows where the `city` column has the value "New York," run the following query:

```sql
SELECT * FROM read_csv('example.csv', header=true) WHERE city = 'New York';
```

### Explanation

- `read_csv('example.csv', header=true)`: Reads the CSV file `example.csv` and considers the first row as headers.
- `WHERE city = 'New York'`: Filters rows where the `city` column is "New York."

### Output

You should see the following filtered result:

```plaintext
+----+------+-----+-----------+
| id | name | age |   city    |
+----+------+-----+-----------+
|  1 | John |  30 | New York  |
|  5 | Eve  |  22 | New York  |
+----+------+-----+-----------+
```

## Step 4: Export the Filtered Data (Optional)

If you want to export the filtered data to another CSV file, you can redirect the output to a file using the `>` operator. For example:

```bash
anyquery -q "SELECT * FROM read_csv('example.csv', header=true) WHERE city = 'New York'" --csv > filtered.csv
```

This command will save the filtered rows to `filtered.csv`.

## Conclusion

You have successfully filtered a CSV file by a column value using Anyquery. This method can be extended to filter by multiple columns, apply complex conditions, and export the results in various formats. For more information on querying files, refer to the [Anyquery documentation](https://anyquery.dev/docs/usage/querying-files).
