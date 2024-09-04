---
title: "How to bulk edit data in a Notion database?"
description: "Learn how to bulk edit data in a Notion database using Anyquery. Follow these steps to configure the Notion plugin, query data, back up, and perform various bulk edits."
---

# How to Bulk Edit Data in a Notion Database using Anyquery

Anyquery is a powerful SQL query engine that allows you to interact with various data sources, including a Notion database. This tutorial will guide you through the steps to bulk edit data in a Notion database using Anyquery.

## Prerequisites

Before starting, ensure you have the following:
- A working installation of Anyquery. For installation instructions, visit [Anyquery Installation](https://anyquery.dev/docs/#installation).
- The Notion plugin installed in Anyquery (`anyquery install notion`). For more details, visit the [Notion integration guide](https://anyquery.dev/integrations/notion).
- Your Notion API key and the database shared with your integration. If you haven't done this, follow the steps in the [Notion plugin documentation](/plugins/notion/README.md).

## Step 1: Configure the Notion Plugin

First, you need to configure the Notion plugin with your API key and the database ID.

1. Install the Notion plugin:
    ```bash
    anyquery install notion
    ```

2. During installation, provide the Notion API key and database ID when prompted. You can find the database ID in the URL of your Notion database. For example, in the URL `https://www.notion.so/myworkspace/My-Database-1234567890abcdef1234567890abcdef`, the database ID is `1234567890abcdef1234567890abcdef`.

## Step 2: Query the Existing Data

To edit data, you first need to query the existing data to understand the schema and existing values. Run the following query to list all records in your Notion database:

```sql
SELECT * FROM notion_database;
```

## Step 3: Backup the Data (Optional but Recommended)

Before making bulk edits, it is a good practice to back up the existing data. You can export the data to a CSV file:

```bash
anyquery -q "SELECT * FROM notion_database" --csv > notion_backup.csv
```

## Step 4: Bulk Edit Data

Now you can perform bulk edits in your Notion database. Here are some examples of common bulk edit operations:

### Example 1: Update Multiple Records

Suppose you want to update the `status` column to 'Completed' for all tasks assigned to 'John Doe':

```sql
UPDATE notion_database
SET status = 'Completed'
WHERE assignee = 'John Doe';
```

### Example 2: Change Due Dates for a Specific Project

To change the due dates for all tasks under a specific project, you can run:

```sql
UPDATE notion_database
SET due_date = '2024-12-31'
WHERE project_name = 'Project Alpha';
```

### Example 3: Add a Tag to Multiple Records

To add a tag 'High Priority' to all tasks that are due in the next week:

```sql
UPDATE notion_database
SET tags = json_insert(tags, '$[#]', 'High Priority')
WHERE due_date BETWEEN date('now') AND date('now', '+7 days');
```

### Example 4: Delete Multiple Records

To delete all records that are marked as 'Archived':

```sql
DELETE FROM notion_database
WHERE status = 'Archived';
```

## Step 5: Verify the Changes

After making the edits, verify the changes by querying the database again:

```sql
SELECT * FROM notion_database;
```

## Conclusion

You have successfully bulk edited data in your Notion database using Anyquery. This approach allows you to efficiently manage and update large volumes of data with SQL. For more details on the Notion plugin and further customization, refer to the [Notion plugin documentation](https://anyquery.dev/integrations/notion).

Happy querying!
