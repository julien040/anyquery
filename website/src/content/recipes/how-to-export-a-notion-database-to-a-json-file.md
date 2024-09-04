---
title: "How to export a Notion database to a JSON file?"
description: "Learn how to export a Notion database to a JSON file using Anyquery. This step-by-step guide covers installation, querying, and exporting data efficiently."
---

# How to Export a Notion Database to a JSON File

Anyquery is a SQL query engine that allows you to query data from various sources, including Notion databases. This tutorial will guide you through exporting a Notion database to a JSON file using Anyquery.

## Prerequisites

Before starting, ensure you have the following:

- A working [installation of Anyquery](https://anyquery.dev/docs/#installation)
- A Notion database that you want to export
- The Notion plugin installed. Follow the [Notion plugin integration guide](https://anyquery.dev/integrations/notion) to set it up.

## Step 1: Install the Notion Plugin

First, install the Notion plugin for Anyquery:

```bash
anyquery install notion
```

Follow the instructions to provide your Notion API key and database ID. If you haven't already, you'll need to create a Notion integration and share the database with that integration.

Refer to the [Notion plugin documentation](https://anyquery.dev/integrations/notion) for detailed instructions on obtaining and using your API key and database ID.

## Step 2: Query the Notion Database

Once the Notion plugin is installed and configured, you can query your Notion database using Anyquery. The database will be accessible as a virtual table named `notion_database`.

Run the following command to list all the records in your Notion database:

```bash
anyquery -q "SELECT * FROM notion_database"
```

## Step 3: Export the Notion Database to a JSON File

To export the Notion database to a JSON file, you need to run a query and specify the output format as JSON. Use the `--json` flag to export the result in JSON format.

Run the following command to export the Notion database to a JSON file:

```bash
anyquery -q "SELECT * FROM notion_database" --json > notion_database.json
```

## Step 4: Verify the JSON File

After running the above command, you should find a file named `notion_database.json` in your current directory. Open the file in a text editor or JSON viewer to verify that it contains the data from your Notion database.

## Additional Tips

### Modifying Columns

You can modify the columns or apply filters to the data before exporting. For example, to export only the `Name` and `Age` columns where `Age` is greater than 25, you can run:

```bash
anyquery -q "SELECT name, age FROM notion_database WHERE age > 25" --json > filtered_notion_database.json
```

### Handling Complex Data Types

Note that arrays and objects in Notion are represented as JSON strings. You can use JSON functions to handle these types. For example, to extract the first element of a multi-select property:

```bash
anyquery -q "SELECT id, json_extract(multi_select_field, '$[0]') AS first_select FROM notion_database" --json > processed_notion_database.json
```

### Caching Data

If you plan to export data multiple times, you can cache the data in a local SQLite database to speed up the process:

```bash
anyquery -q "CREATE TABLE cached_notion_data AS SELECT * FROM notion_database"
anyquery -q "SELECT * FROM cached_notion_data" --json > cached_notion_database.json
```

## Conclusion

You have successfully exported a Notion database to a JSON file using Anyquery. This powerful tool allows you to query and manipulate data from various sources with SQL. Explore other features of Anyquery to enhance your data processing workflows.

For more information, refer to the [Anyquery documentation](https://anyquery.dev/docs/usage/*) and the [Notion plugin documentation](https://anyquery.dev/integrations/notion).
