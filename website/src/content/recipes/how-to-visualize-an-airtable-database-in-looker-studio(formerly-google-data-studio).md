---
title: "How to visualize an Airtable database in Looker Studio(formerly Google Data Studio)?"
description: "Learn to visualize an Airtable database in Looker Studio using Anyquery. Follow steps to install plugins, set up connections, run SQL queries, and create visualizations."
---

# How to Visualize an Airtable Database in Looker Studio

Anyquery allows you to write SQL queries on pretty much any data source, including Airtable. This tutorial guides you through visualizing an Airtable database in Looker Studio (formerly Google Data Studio) using Anyquery. 

## Prerequisites

Before starting, ensure you have the following:
- A working installation of Anyquery. Follow the installation instructions [here](https://anyquery.dev/docs/#installation).
- An Airtable account and a created table in Airtable. Ensure the schema is correctly defined.
- Looker Studio access with a Google account.

## Step 1: Install and Configure the Airtable Plugin

First, install the Airtable plugin:

```bash
anyquery install airtable
```

You will be prompted to provide the following details:
1. **Airtable API Key**: Create an API key from the [Airtable account page](https://airtable.com/account).
2. **Base ID**: Find it in the URL of your Airtable base.
3. **Table Name**: The table name you want to visualize.
4. **Enable Cache**: Choose whether to enable caching for faster queries.

## Step 2: Start the Anyquery Server

Launch the Anyquery server:

```bash
anyquery server
```

Because Looker Studio is a web-based tool, and `anyquery` binds locally, you need to expose the server to the internet. You can use a tool like [ngrok](https://ngrok.com/) to create a secure tunnel to your local server.

```bash
ngrok tcp 8070
```

Copy the forwarding URL (e.g., `tcp://0.tcp.ngrok.io:12345`) and use it as the hostname in the next step.

## Step 3: Connect Looker Studio

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
6. Selecting a table does not work in Looker Studio. You can only run SQL queries by clicking on the "Personalized query" tab and typing your query. Example: `SELECT * FROM airtable_table;`.
7. Click on the `Authenticate` button to verify that the connection is successful.

:::caution
Looker Studio does not handle all column names well. If you have a column name with a space or special character, you might not be able to pick it up as a dimension or measure. You can rename the column in the query to work around this issue. For example, `SELECT \`Column Name\` AS \`column_name\` FROM airtable_table;`.
:::

## Example of a Query

Here is an example query to list all records from your Airtable table:

```sql
SELECT * FROM airtable_table;
```

You can also filter and transform the data as needed. For instance, to filter records based on a specific condition, use:

```sql
SELECT * FROM airtable_table WHERE status = 'Active';
```

## Creating Your First Visualization

1. After running your query, click on the `Add to Report` button.
2. Create visualizations by dragging desired fields to the canvas and customizing the charts and tables as needed.

## Conclusion

You have successfully connected Looker Studio to Anyquery and visualized your Airtable database. Now you can explore and create meaningful visualizations using Looker Studio.

For more information on Airtable queries, refer to the [Airtable plugin documentation](https://anyquery.dev/integrations/airtable). For troubleshooting common issues, visit the [troubleshooting guide](https://anyquery.dev/docs/usage/troubleshooting/).
