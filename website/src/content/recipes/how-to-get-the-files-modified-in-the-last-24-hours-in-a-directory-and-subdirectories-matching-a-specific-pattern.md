---
title: "How to get the files modified in the last 24 hours in a directory and subdirectories matching a specific pattern?"
description: "Learn to find files modified in the last 24 hours using Anyquery. This guide covers installation, setup, and querying techniques to filter files based on patterns and modification time."
---

# How to Get the Files Modified in the Last 24 Hours in a Directory and Subdirectories Matching a Specific Pattern

Anyquery is a SQL query engine that allows you to run SQL queries on pretty much anything. In this tutorial, we will learn how to get the files modified in the last 24 hours in a directory and its subdirectories, matching a specific pattern.

## Prerequisites

Before starting, ensure you have the following:
- A working installation of Anyquery. Refer to the [installation guide](https://anyquery.dev/docs/#installation).
- The file plugin installed. You can install it by running:

```bash
anyquery install file
```

## Step-by-Step Guide

1. **Install and Setup Anyquery:**

Ensure Anyquery and the file plugin are installed on your system:

```bash
anyquery install file
```

2. **Query Modified Files:**

To get the files modified in the last 24 hours in a directory and its subdirectories, matching a specific pattern, we will use the `file_search` function. This function allows us to search for files matching a pattern and use SQL functions to filter files based on their modification time.

### Example Query

For example, let's assume we want to find all `.txt` files in the directory `/path/to/dir` that were modified in the last 24 hours:

```sql
SELECT * FROM file_search('/path/to/dir/*.txt')
WHERE last_modified > datetime('now', '-1 day');
```

### Explanation
- `file_search('/path/to/dir/*.txt')`: This part of the query uses the `file_search` function to search for all `.txt` files in the specified directory and its subdirectories.
- `WHERE last_modified > datetime('now', '-1 day')`: This condition filters the results to include only the files that were modified in the last 24 hours.

### Running the Query

Open your terminal and run the following command to execute the query:

```bash
anyquery -q "SELECT * FROM file_search('/path/to/dir/*.txt') WHERE last_modified > datetime('now', '-1 day');"
```

### Output

The output will list all `.txt` files in `/path/to/dir` and its subdirectories that were modified in the last 24 hours, along with their details such as path, file name, file type, size, last modified time, and whether they are directories.

### Further Customization

You can further customize the query to match different patterns or directories. For instance, to search for `.log` files:

```sql
SELECT * FROM file_search('/path/to/dir/*.log')
WHERE last_modified > datetime('now', '-1 day');
```

## Conclusion

You have successfully learned how to get the files modified in the last 24 hours in a directory and its subdirectories, matching a specific pattern using Anyquery. For more information, refer to the [file plugin documentation](https://anyquery.dev/docs/usage/querying-files) and the [official Anyquery documentation](https://anyquery.dev/docs/).
