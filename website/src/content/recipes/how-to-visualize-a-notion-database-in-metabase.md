---
title: "How to visualize a Notion database in Metabase?"
description: "Learn to visualize Notion databases in Metabase using Anyquery. Follow steps to install, configure, connect, and create visualizations for insightful data analysis."
---

# How to Visualize a Notion Database in Metabase

In this tutorial, we will explore how to visualize a Notion database in Metabase using Anyquery. Anyquery is a SQL query engine that allows you to run SQL queries on various data sources, including Notion databases. Metabase is a powerful business intelligence tool that allows you to create and share data visualizations.

## Prerequisites

Before starting, ensure you have the following:

- A working [installation of Anyquery](https://anyquery.dev/docs/#installation)
- [Metabase](https://www.metabase.com/docs/latest/installation-and-operation/running-metabase-on-docker) running
- A Notion database with the correct schema

Ensure you have set up your Notion database correctly. For detailed setup, refer to the [Notion integration guide](https://anyquery.dev/integrations/notion).

## Step 1: Install and Configure Anyquery

### 1.1 Install Anyquery

If you haven't installed Anyquery yet, follow the installation instructions provided [here](https://anyquery.dev/docs/#installation).

### 1.2 Install the Notion Plugin

Install the Notion plugin for Anyquery:

```bash
anyquery install notion
```

### 1.3 Configure the Notion Plugin

Follow the [Notion integration guide](https://anyquery.dev/integrations/notion) to get your Notion API key and database ID. When prompted during the plugin installation, provide the API key and database ID.

```bash
anyquery profiles new default notion my_notion_profile
```

## Step 2: Set Up the Anyquery Server

Launch the Anyquery server:

```bash
anyquery server
```

Since Metabase is a web-based tool, and Anyquery binds locally, you need to expose the Anyquery server to the internet. Use a tool like [ngrok](https://ngrok.com/) to create a secure tunnel to your local server:

```bash
ngrok tcp 8070
```

Copy the forwarding URL (e.g., `tcp://0.tcp.ngrok.io:12345`) and use it as the hostname in the next step.

## Step 3: Connect Metabase to Anyquery

1. Open Metabase in your browser and go to the database settings.
   - URL: `https://{your-metabase-url}/admin/databases/create`
2. Select MySQL as the database type.
3. Fill in the following details:
   - **Name**: A memorable name for the connection.
   - **Host**: The forwarding URL from ngrok (e.g., `0.tcp.ngrok.io`).
   - **Port**: The port from ngrok (e.g., `12345`).
   - **Database name**: `main`.
   - **Username**: Set `root` unless you have set an [auth-file](https://anyquery.dev/docs/usage/mysql-server#adding-authentication).
   - **Password**: Leave it empty unless you have set an [auth-file](https://anyquery.dev/docs/usage/mysql-server#adding-authentication).
4. Click on the `Save` button to verify that the connection is successful.

![Metabase Connection](/images/docs/Ws1UhIKV.png)

## Step 4: Create a Model in Metabase

Go back to the Metabase dashboard and create a new model with a native query. Click on the `+ New` button on the top right and select `Model`, then `Use a native query`.

```sql
-- List all rows from your Notion database
SELECT * FROM my_notion_profile_notion_database;
```

Click on the ▶️ button (or run `⌘ + enter`) to run the query. If it works, click on the `Save` button to save the model (you can name it according to your preference, e.g., `Notion Database`).

![Metabase Model](/images/docs/GgCl8quP.png)

## Step 5: Create Visualizations

Now, create a new question and visualize the data. Click on the `+ New` button on the top right and select `Question`. You can now select the model you created and start building your visualization.

Once you have created your questions, you can create a dashboard to visualize the data. Click on the `+ New` button on the top right and select `Dashboard`. You can now add the questions you created to the dashboard.

For example, here is a dashboard showing data from a Notion database:

![Metabase Dashboard](/images/docs/mfjW79d8.png)

## Conclusion

You have successfully connected Metabase to Anyquery and visualized data from a Notion database. Now you can explore and visualize data from any source using Metabase. For more information, refer to the [Metabase integration guide](https://anyquery.dev/integrations/metabase) and the [Notion integration guide](https://anyquery.dev/integrations/notion).
