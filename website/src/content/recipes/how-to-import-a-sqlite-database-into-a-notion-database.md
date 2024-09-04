---
title: "How to import a SQLite database into a Notion database?"
description: "Learn how to import a SQLite database into Notion using Anyquery. This guide covers setting up Notion plugin, exporting data from SQLite, and importing it into Notion."
---

# How to Import a SQLite Database into a Notion Database

Anyquery allows you to query data from different sources like databases, APIs, and even files. In this tutorial, we will guide you on how to import a SQLite database into a Notion database using Anyquery.

## Prerequisites

Before starting, ensure you have the following:

- A working [installation of Anyquery](https://anyquery.dev/docs/#installation)
- A Notion account and the Notion plugin installed in Anyquery

## Step 1: Set up the Notion Plugin

Firstly, you'll need to install and set up the Notion plugin.

### Install the Notion Plugin

Open your terminal and run:

```bash
anyquery install notion
```

### Setup the Notion API Key

1. Go to [Notion My Integrations](https://www.notion.so/my-integrations) and create a new integration.
2. Fill in the form and save the integration.
3. Copy the `Internal Integration Token`.

### Share Your Database with the Integration

1. Open the Notion workspace with the database you want to import data into.
2. Click on the three dots in the top right corner of your database.
3. Select `+ Add Connections` and add your created integration.

### Configure the Plugin in Anyquery

In your terminal:

```bash
anyquery profile new default notion my_notebook
```

Fill in the `Internal Integration Token` and the database ID (found in the URL of your database).

## Step 2: Export Data from SQLite

You need to have the data you want to import into Notion in a SQLite database.

### Export Data to a Temporary SQLite Table

```sql
CREATE TABLE temp_data AS SELECT * FROM your_table;
```

## Step 3: Import Data into Notion Database

### Query to Insert Data

Run the following SQL command to import your data into the Notion database:

```bash
anyquery -q "INSERT INTO my_notebook (column1, column2, ...) SELECT column1, column2, ... FROM temp_data"
```

Ensure that the columns in your Notion database match those in your SQLite database. You may need to map columns appropriately.

### Example Query

For example, if you have a SQLite table `employees` with columns `name`, `email`, and `position`, and a Notion database with the same columns, the query would look like this:

```bash
anyquery -q "INSERT INTO my_notebook (name, email, position) SELECT name, email, position FROM temp_data"
```

## Conclusion

You have successfully imported data from a SQLite database into a Notion database using Anyquery. Now you can explore and manipulate this data within Notion.

### Additional Resources

- [Notion Plugin Documentation](https://anyquery.dev/integrations/notion)
- [Anyquery Troubleshooting](https://anyquery.dev/docs/usage/troubleshooting/)

By following these steps, you can integrate data from various sources into Notion, making it easier to manage and visualize your data.
