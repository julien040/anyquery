---
title: "How to bulk delete data in a Notion database?"
description: "Learn how to bulk delete data in a Notion database using Anyquery. Follow steps to install, set up, and execute efficient SQL queries for data management."
---

# How to Bulk Delete Data in a Notion Database

Anyquery is a SQL query engine that enables you to execute SQL queries on virtually anything, including Notion databases. In this tutorial, we will cover how to bulk delete data in a Notion database using Anyquery.

## Prerequisites

Before starting, ensure you have the following:
- A working [installation of Anyquery](https://anyquery.dev/docs/#installation)
- The Notion plugin installed: `anyquery install notion`
- Your Notion API key and Database ID.

### Setting Up Notion Plugin

1. Go to [Notion's My Integrations page](https://www.notion.so/my-integrations) and create a new integration.
2. Select the relevant workspace and copy the integration token.
3. Share the database you want to query with the integration.

Refer to the [Notion integration guide](https://anyquery.dev/integrations/notion) for more details.

## Step 1: Launch Anyquery

First, start Anyquery in shell mode:

```bash
anyquery
```

## Step 2: List the Data in the Notion Database

Before we delete data, it's a good idea to list the data you want to delete. Here is an example query to select all rows from a Notion database:

```sql
SELECT * FROM notion_database;
```

Replace `notion_database` with the actual name of your Notion database.

## Step 3: Bulk Delete Data

To delete specific rows in bulk, use the `DELETE FROM` statement with appropriate conditions. For example, to delete all rows where the column `status` is `completed`:

```sql
DELETE FROM notion_database WHERE status = 'completed';
```

### Example Workflow

Here's a complete workflow:

1. Start the Anyquery shell:

    ```bash
    anyquery
    ```

2. List all rows to ensure you are deleting the correct data:

    ```sql
    SELECT * FROM notion_database;
    ```

3. Delete all rows where the status is `completed`:

    ```sql
    DELETE FROM notion_database WHERE status = 'completed';
    ```

4. Verify the deletion by listing the remaining rows:

    ```sql
    SELECT * FROM notion_database;
    ```

## Note

- Deleting data using Anyquery will move the items to the trash in Notion. They can be restored from the Notion UI if needed.
- Ensure you have the correct permissions and a backup of important data before performing bulk deletions.

## Conclusion

You have successfully learned how to bulk delete data in a Notion database using Anyquery. Use this process to efficiently manage and clean up your Notion databases. For more information, refer to the [Notion integration guide](https://anyquery.dev/integrations/notion) and [Anyquery documentation](https://anyquery.dev/docs/usage/*).
