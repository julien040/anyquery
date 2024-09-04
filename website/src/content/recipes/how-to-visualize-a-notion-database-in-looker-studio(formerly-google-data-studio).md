---
title: "How to visualize a Notion database in Looker Studio(formerly Google Data Studio)?"
description: "Learn to visualize your Notion database in Looker Studio with Anyquery. Install plugins, query data, set up secure connections, and create custom visualizations seamlessly."
---

# Visualize a Notion Database in Looker Studio (Google Data Studio)

In this tutorial, we will guide you through the process of visualizing a Notion database in Looker Studio (formerly Google Data Studio) using Anyquery. This involves querying your Notion database with Anyquery and visualizing the data in Looker Studio.

## Prerequisites

Before starting, ensure you have the following:

- A working installation of Anyquery. See [installation guide](https://anyquery.dev/docs/#installation).
- A Notion account with a database you want to visualize.
- A Google account to access Looker Studio.

## Step 1: Set Up Anyquery and Notion

### Install Anyquery and the Notion Plugin

First, install Anyquery and the Notion plugin. Follow the [Notion plugin integration guide](https://anyquery.dev/integrations/notion) to authenticate your Notion account and get the necessary credentials and database ID.

```bash
anyquery install notion
```

During setup, you will be prompted to provide your Notion API key and the Database ID of the Notion database you want to query.

### Query the Notion Database

Ensure Anyquery can access your Notion database by running a query:

```sql
SELECT * FROM notion_database;
```

Replace `notion_database` with the name of your Notion database.

## Step 2: Start Anyquery Server

Launch the Anyquery server which will act as a MySQL server:

```bash
anyquery server
```

## Step 3: Expose Anyquery Server to the Internet

Because Looker Studio is a web-based tool, and Anyquery binds locally, you need to expose the server to the internet. You can use a tool like [ngrok](https://ngrok.com/) to create a secure tunnel to your local server.

```bash
ngrok tcp 8070
```

Copy the forwarding URL (e.g., `tcp://0.tcp.ngrok.io:12345`) and use it as the hostname in the next step.

## Step 4: Connect Looker Studio

1. Open [Looker Studio](https://lookerstudio.google.com/u/0/navigation/reporting) in your browser.
2. Click on the `+` icon (empty report) to create a new report.
3. In the search bar, type `MySQL` and select the `MySQL` connection.
4. Authorize Looker Studio to access your data.
5. Fill in the following details:
   - **Host**: The forwarding URL from ngrok (e.g., `0.tcp.ngrok.io`).
   - **Port**: The port from ngrok (e.g., `12345`).
   - **Database**: `main`.
   - **Username**: Set `root` unless you have set an [auth-file](https://anyquery.dev/docs/usage/mysql-server#adding-authentication).
   - **Password**: Leave it empty unless you have set an [auth-file](https://anyquery.dev/docs/usage/mysql-server#adding-authentication).
6. Selecting a table does not work in Looker Studio. You can only run SQL queries by clicking on the "Personalized query" tab and typing your query. For example:

```sql
SELECT * FROM notion_database;
```

7. Click on the `Authenticate` button to verify that the connection is successful.

:::caution
Looker Studio does not handle all column names well. If you have a column name with a space or special characters, you might not be able to pick it up as a dimension or measure. You can rename the column in the query to work around this issue. For example:

```sql
SELECT `Column With Spaces` AS `ColumnWithSpaces` FROM notion_database;
```
:::

## Step 5: Create Visualizations in Looker Studio

Once connected, you can create various visualizations in Looker Studio using the data from your Notion database.

1. Drag and drop the fields from your Notion database to the canvas to create charts and tables.
2. Use the filtering and styling options in Looker Studio to customize the visualizations as per your requirements.
3. Save and share your report with others.

## Example Visualization

For example, you can create a bar chart to visualize the task statuses in your Notion database:

1. Drag the `Status` field to the `Dimension` section.
2. Drag the `Count` field to the `Metric` section.
3. Customize the chart as needed and save your report.

![Looker Studio Dashboard Example](https://anyquery.dev/images/docs/oE0a8dtb.png)

## Conclusion

You have successfully connected Looker Studio to Anyquery and visualized your Notion database. Now, you can explore and visualize data from any source using Looker Studio. For more information, refer to the [Looker Studio integration guide](https://anyquery.dev/integrations/looker-studio).

For troubleshooting, visit the [Anyquery troubleshooting guide](https://anyquery.dev/docs/usage/troubleshooting/).
