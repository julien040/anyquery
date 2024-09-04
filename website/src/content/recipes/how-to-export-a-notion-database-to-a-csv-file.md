---
title: "How to export a Notion database to a CSV file?"
description: "Learn how to export your Notion database to a CSV file using Anyquery. Follow step-by-step instructions from setting up Notion integration to running SQL queries."
---

# How to Export a Notion Database to a CSV File

Anyquery is a powerful SQL query engine that enables you to run SQL queries on various data sources, including Notion databases. In this tutorial, we'll walk you through the steps to export a Notion database to a CSV file using Anyquery.

## Prerequisites

Before starting, ensure you have the following:

- A working [installation of Anyquery](https://anyquery.dev/docs/#installation)
- [Configured Notion integration](https://anyquery.dev/integrations/notion)

To set up the Notion integration, follow these steps:

1. **Create an Integration in Notion**:
    - Go to the [Notion My Integrations page](https://www.notion.so/my-integrations).
    - Click on `+ New integration` and create a new integration.
    - Note the `Integration Token`.

2. **Share Your Database with the Integration**:
    - Open the database in Notion.
    - Share the database with the integration you created.

3. **Install the Notion Plugin in Anyquery**:
    ```bash
    anyquery install notion
    ```

4. **Configure the Notion Plugin**:
    - Run Anyquery and follow the prompts to enter the integration token and the database ID.

## Exporting Notion Database to CSV

Once the Notion integration is set up, you can easily query your Notion database and export the results to a CSV file. Follow these steps:

1. **Open Anyquery Shell**:
    ```bash
    anyquery
    ```

2. **Run SQL Query on Notion Database**:
    - Query the Notion database to retrieve the desired data. Replace `notion_database` with the name of your Notion database in Anyquery:
    ```sql
    SELECT * FROM notion_database;
    ```

3. **Export to CSV**:
    - Exit the shell mode and run the following command to export the query result to a CSV file:
    ```bash
    anyquery -q "SELECT * FROM notion_database" --csv > notion_database.csv
    ```

## Example

Here’s a complete example of exporting a Notion database named `my_tasks` to a CSV file:

1. **Open Anyquery Shell**:
    ```bash
    anyquery
    ```

2. **Query the `my_tasks` Database**:
    ```sql
    SELECT * FROM my_tasks;
    ```

3. **Export to CSV**:
    ```bash
    anyquery -q "SELECT * FROM my_tasks" --csv > my_tasks.csv
    ```

## Conclusion

You've successfully exported your Notion database to a CSV file using Anyquery. This tutorial covered the essential steps from setting up the Notion integration to running SQL queries and exporting the results. For more information on querying and exporting data, check out the [Anyquery documentation](https://anyquery.dev/docs/usage/querying-files) and [Notion integration guide](https://anyquery.dev/integrations/notion).
