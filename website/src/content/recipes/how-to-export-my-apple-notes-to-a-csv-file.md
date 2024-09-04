---
title: "How to export my Apple Notes to a CSV file?"
description: "Export your Apple Notes to a CSV file using Anyquery's SQL query engine. Install the Apple Notes plugin, run SQL queries, and export your notes effortlessly."
---

# How to Export Apple Notes to a CSV File

Anyquery is a SQL query engine that enables you to execute SQL queries on virtually anything, including Apple Notes. This tutorial will guide you through the steps to export your Apple Notes to a CSV file using Anyquery.

## Prerequisites

Before you start, ensure you have the following:
- A working [installation of Anyquery](https://anyquery.dev/docs/#installation)
- The [Apple Notes plugin](https://anyquery.dev/integrations/notes) installed

## Step 1: Install the Apple Notes Plugin

First, you need to install the Apple Notes plugin. Open your terminal and run the following command:

```bash
anyquery install notes
```

On the first query, a popup will ask if you want your terminal to have access to Apple Notes. Make sure to allow it.

## Step 2: Query Your Notes

Once the plugin is installed and configured, you can query your Apple Notes. Here is an example query to get all your notes:

```sql
SELECT * FROM notes_items;
```

## Step 3: Export Notes to a CSV File

To export your notes to a CSV file, use the following command in your terminal:

```bash
anyquery -q "SELECT * FROM notes_items" --csv > notes.csv
```

### Example of Exporting Notes with Specific Columns

You can also select specific columns to export. For instance, to export only the note title and creation date, use the following command:

```bash
anyquery -q "SELECT name AS 'Title', creation_date AS 'Created At' FROM notes_items" --csv > notes.csv
```

## Step 4: Handling Large Outputs

If you have a lot of notes, the default output mode might not be suitable. Switch to a different output format like JSON, plain text, etc., to avoid issues:

```bash
anyquery -q "SELECT * FROM notes_items" --json > notes.json
```

## Conclusion

You have successfully exported your Apple Notes to a CSV file using Anyquery. You can further manipulate or share this CSV file as needed.

For more information on available functions and features, refer to the [functions documentation](https://anyquery.dev/docs/reference/functions). If you encounter any issues, check the [troubleshooting guide](https://anyquery.dev/docs/usage/troubleshooting/).
