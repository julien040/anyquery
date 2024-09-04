---
title: "How to import a CSV file into a Notion database?"
description: "Learn to import CSV files into Notion databases using Anyquery. Get prerequisites, install plugins, set up schemas, and execute SQL commands for seamless data transfer."
---

# How to Import a CSV File into a Notion Database

Anyquery is a SQL query engine that enables you to execute SQL queries on virtually anything, including CSV files and Notion databases. In this tutorial, we'll walk through the steps to import a CSV file into a Notion database using Anyquery.

## Prerequisites

Before starting, ensure you have the following:
- A working [installation of Anyquery](https://anyquery.dev/docs/#installation)
- The [Notion plugin installed](https://anyquery.dev/integrations/notion)
- The [CSV plugin installed](https://anyquery.dev/integrations/csv)
- A Notion database with the correct schema to receive the CSV data

## Step 1: Install the Plugins

First, install the necessary plugins if they are not already installed.

### Notion Plugin

```bash
anyquery install notion
```

Follow the instructions to provide your Notion API key and database ID. You can find these details in the [Notion plugin documentation](https://anyquery.dev/integrations/notion).

## Step 2: Set Up the Notion Database

Ensure that your Notion database has the correct schema to match the CSV file you want to import. The column names in the CSV file should correspond to the property names in the Notion database.

For this example, let's assume your CSV file has the following structure:

```csv
Name, Age, Email
Alice, 30, alice@example.com
Bob, 24, bob@example.com
```

Your Notion database should have properties named `Name`, `Age`, and `Email`.

## Step 3: Import the CSV File into Notion

Once the plugins are installed and the Notion database is set up, you can run the following SQL query to import the CSV file into the Notion database.

Replace `[path/to/yourfile.csv]` with the path to your CSV file. Replace `notion_database` with the name of your Notion database.

```bash
anyquery -q "INSERT INTO notion_database (Name, Age, Email) SELECT Name, Age, Email FROM read_csv('[path/to/yourfile.csv]', header=true)"
```

### Example

If your CSV file is located at `/path/to/contacts.csv`, the command would be:

```bash
anyquery -q "INSERT INTO notion_database (Name, Age, Email) SELECT Name, Age, Email FROM read_csv('/path/to/contacts.csv', header=true)"
```

## Additional Tips

- **Data Types:** Ensure that the data types in the CSV file match the expected data types in the Notion database.
- **Error Handling:** If you encounter any errors, check the CSV file and Notion database schema for consistency.
- **Cache Clearing:** If you need to refresh the data, you can clear the plugin cache using `SELECT clear_plugin_cache('csv');` and `SELECT clear_plugin_cache('notion');`.

## Conclusion

You have successfully imported a CSV file into a Notion database using Anyquery. Now you can easily transfer data between CSV files and your Notion databases. For more advanced usage and troubleshooting, refer to the [Anyquery documentation](https://anyquery.dev/docs/usage/troubleshooting/).
