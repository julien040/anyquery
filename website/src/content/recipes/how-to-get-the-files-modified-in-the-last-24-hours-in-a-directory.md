---
title: "How to get the files modified in the last 24 hours in a directory?"
description: "Learn to identify files modified in the last 24 hours in a directory using Anyquery. Follow step-by-step instructions to filter, run queries, and export results to JSON or CSV."
---

# How to Get the Files Modified in the Last 24 Hours in a Directory

**Anyquery** is a SQL query engine that allows you to run SQL queries on various data sources, including files. In this tutorial, we'll learn how to get the files modified in the last 24 hours in a directory using Anyquery.

## Prerequisites

Before starting, ensure you have the following:
- A working installation of Anyquery. Follow the [installation guide](https://anyquery.dev/docs/#installation) if you don't have it installed.
- The `file` plugin installed. You can install it by running the following command:
    ```bash
    anyquery install file
    ```

## Step-by-Step Guide

### Step 1: List the Files in a Directory

First, let's list all the files in a directory. Use the `file_list` function provided by the file plugin.

```sql
SELECT * FROM file_list('/path/to/your/directory');
```

Replace `/path/to/your/directory` with the actual path to your directory.

### Step 2: Filter Files Modified in the Last 24 Hours

To filter files modified in the last 24 hours, we will use the `datetime` function to compare the `last_modified` column. SQLite's `datetime` function can be used to get the current date and time and subtract 24 hours from it.

```sql
SELECT * FROM file_list('/path/to/your/directory')
WHERE last_modified > datetime('now', '-1 day');
```

### Step 3: Run the Query

Run the query using Anyquery to get the list of files modified in the last 24 hours.

```bash
anyquery -q "SELECT * FROM file_list('/path/to/your/directory') WHERE last_modified > datetime('now', '-1 day');"
```

### Step 4: Export the Result

You can export the result to different formats like JSON or CSV for further analysis or sharing.

#### Export to JSON

```bash
anyquery -q "SELECT * FROM file_list('/path/to/your/directory') WHERE last_modified > datetime('now', '-1 day');" --json > modified_files.json
```

#### Export to CSV

```bash
anyquery -q "SELECT * FROM file_list('/path/to/your/directory') WHERE last_modified > datetime('now', '-1 day');" --csv > modified_files.csv
```

### Summary

By following these steps, you can easily get the files modified in the last 24 hours in a directory using Anyquery. You can also filter, export, and manipulate the data based on your needs.

For more information on the `file` plugin and other available functions, refer to the [file plugin documentation](https://anyquery.dev/integrations/file).
