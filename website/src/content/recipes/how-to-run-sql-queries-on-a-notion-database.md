---
title: "How to run SQL queries on a Notion database?"
description: "Learn to run SQL queries on Notion databases using Anyquery. This tutorial covers plugin installation, API key setup, database ID retrieval, and executing CRUD operations."
---

# How to Run SQL Queries on a Notion Database

Anyquery is a SQL query engine that allows you to run SQL queries on pretty much any data source, including Notion databases. This tutorial will guide you through the steps to query a Notion database using SQL.

## Introduction

Anyquery allows you to query data from Notion databases using SQL. You can read, insert, update, and delete records in a Notion database. This can be useful for generating reports, transforming data, or integrating Notion data with other tools.

Before starting, ensure you have a working installation of Anyquery. If not, refer to the [installation guide](https://anyquery.dev/docs/#installation).

## Step 1: Install the Notion Plugin

To begin, you need to install the Notion plugin for Anyquery. Open your terminal and run:

```bash
anyquery install notion
```

## Step 2: Get Your Notion API Key

1. Go to [Notion's My Integrations page](https://www.notion.so/my-integrations).
2. Click on the `+ New integration` button.
3. Fill in the form with the following information:
    - Name: Whatever you want
    - Associated workspace: The workspace you want the plugin to have access to
    - Type: Internal
4. Click on the `Save` button and then on `Configure integration settings`.
5. Copy the `token` and paste it when asked by the plugin.
6. Share each database you want to query with the integration you just created:
    - Open the database you want to share.
    - Click on the three dots in the top right corner.
    - Scroll down, hover over `Connect to` and click on the integration you just created.

## Step 3: Find the Database ID

To query a Notion database, you need to find its database ID. You can find it in the URL of the database. For example, if the URL of the database is `https://www.notion.so/myworkspace/My-Database-1234567890abcdef1234567890abcdef`, the database ID is `1234567890abcdef1234567890abcdef`.

## Step 4: Set Up the Connection

During the plugin installation, Anyquery will ask for the Notion API token and the database ID:

```bash
anyquery install notion
```

Provide the required information when prompted.

## Step 5: Run SQL Queries

After setting up the connection, you can run SQL queries on your Notion database. Here are some examples:

### Select All Records

```sql
SELECT * FROM notion_database;
```

### Filter Records

```sql
SELECT * FROM notion_database WHERE name = 'John Doe';
```

### Insert a Record

```sql
INSERT INTO notion_database (name, age) VALUES ('Alice', 30);
```

### Update a Record

```sql
UPDATE notion_database SET age = 31 WHERE name = 'Alice';
```

### Delete a Record

```sql
DELETE FROM notion_database WHERE name = 'Alice';
```

## Schema Information

The schema of the Notion database in Anyquery will match the properties of your Notion database. Common columns include:
- Text
- Number
- Date
- Checkbox
- Select
- Multi-select
- Email
- URL
- Phone
- Formula (read-only)

For more information on the schema, refer to the [Notion plugin documentation](https://anyquery.dev/integrations/notion).

## Conclusion

You have successfully connected Anyquery to a Notion database and run SQL queries on it. Now you can leverage SQL to analyze and manipulate data in your Notion databases. For any advanced usage or troubleshooting, refer to the [troubleshooting guide](https://anyquery.dev/docs/usage/troubleshooting/).
