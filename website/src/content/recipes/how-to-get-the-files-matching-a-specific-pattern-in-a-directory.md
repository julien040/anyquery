---
title: "How to get the files matching a specific pattern in a directory?"
description: "Learn how to use Anyquery to find files matching specific patterns in a directory. Install the `file` plugin, run `file_search` queries, and export results to CSV or JSON."
---

# How to Get Files Matching a Specific Pattern in a Directory

Anyquery is a SQL query engine that allows you to run SQL queries on various data sources, including files in a directory. In this tutorial, we will guide you on how to get files matching a specific pattern in a directory using Anyquery.

## Prerequisites

Before starting, ensure you have the following:
- A working [installation of Anyquery](https://anyquery.dev/docs/#installation).

## Step 1: Install the File Plugin

First, install the `file` plugin in Anyquery. This plugin enables you to query files in a directory with SQL.

```bash
anyquery install file
```

## Step 2: Query Files Matching a Specific Pattern

To get files matching a specific pattern in a directory, you can use the `file_search` table function. This function takes a pattern as an argument and returns files matching that pattern.

### Example Query

Let's say you want to find all `.txt` files in the `/path/to/directory` directory. You can run the following SQL query:

```sql
SELECT * FROM file_search('/path/to/directory/*.txt');
```

Alternatively, you can use the `file_search` function in the shell mode:

```bash
anyquery
```

```sql
SELECT * FROM file_search('/path/to/directory/*.txt');
```

### Filtering Files

You can further filter the results using SQL `WHERE` clauses:

```sql
-- Get .txt files larger than 1MB
SELECT * FROM file_search('/path/to/directory/*.txt') WHERE size > 1048576;

-- Get .log files modified in the last 7 days
SELECT * FROM file_search('/path/to/directory/*.log') WHERE last_modified > datetime('now', '-7 days');
```

See the [functions documentation](https://anyquery.dev/docs/reference/functions) for more information on the available functions.

## Step 3: Exporting Results

You can also export the results to various formats like JSON, CSV, or plain text.

### Export to CSV

```bash
anyquery -q "SELECT * FROM file_search('/path/to/directory/*.txt')" --csv > files.csv
```

### Export to JSON

```bash
anyquery -q "SELECT * FROM file_search('/path/to/directory/*.txt')" --json > files.json
```

Refer to the [exporting results documentation](https://anyquery.dev/docs/usage/exporting-results) for more information on exporting data.

## Schema

The `file_search` table has the following schema:

| Column index | Column name     | type    |
| ------------ | --------------- | ------- |
| 0            | path            | TEXT    |
| 1            | file_name       | TEXT    |
| 2            | file_type       | TEXT    |
| 3            | size            | INTEGER |
| 4            | last_modified   | INTEGER |
| 5            | is_directory    | INTEGER |

## Conclusion

Using Anyquery and the `file` plugin, you can easily query and filter files in a directory based on specific patterns. This tutorial covered the installation of the `file` plugin, querying files with a specific pattern, and exporting the results.

For more information, refer to the [Anyquery documentation](https://anyquery.dev/docs/usage/querying-files).
