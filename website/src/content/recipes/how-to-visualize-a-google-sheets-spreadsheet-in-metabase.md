---
title: "How to visualize a Google Sheets spreadsheet in Metabase?"
description: "Learn to connect Anyquery to Google Sheets and visualize data in Metabase. Follow steps to install plugins, configure connections, and create visualizations using SQL."
---

# How to Visualize a Google Sheets Spreadsheet in Metabase

Anyquery is a SQL query engine that allows you to run SQL queries on various data sources, including Google Sheets. In this tutorial, we will show you how to connect Anyquery to a Google Sheets spreadsheet and visualize it using Metabase.

## Prerequisites

Before starting, ensure you have the following:

- A working [installation of Anyquery](https://anyquery.dev/docs/#installation)
- A Google Sheets spreadsheet
- [Metabase](https://www.metabase.com/docs/latest/installation-and-operation/running-metabase-on-docker) running on Docker

## Step 1: Set Up the Google Sheets Connection

First, you need to install and configure the Google Sheets plugin for Anyquery.

### Install the Google Sheets Plugin

Run the following command to install the plugin:

```bash
anyquery install google_sheets
```

### Authenticate with Google

1. Go to the [Google Cloud Console](https://console.cloud.google.com/), create a new project, and enable the Google Sheets API.
2. Create OAuth 2.0 credentials and note down the `Client ID` and `Client Secret`.
3. Go to [Google Sheets integration](https://integration.anyquery.dev/google-sheets) and fill in the `Client ID` and `Client Secret`.
4. Follow the authentication steps and copy the token provided.

### Configure the Plugin

Run the following command to configure the plugin:

```bash
anyquery profile new default google_sheets
```

You will be prompted to enter the token, client ID, client secret, and the Google Sheets spreadsheet ID. You can find the spreadsheet ID in the URL of the spreadsheet (e.g., in `https://docs.google.com/spreadsheets/d/1D_x7DNwbI9ZOSFjII6BvttPzdLJAymrQwQcOvnHzW9c/edit`, the spreadsheet ID is `1D_x7DNwbI9ZOSFjII6BvttPzdLJAymrQwQcOvnHzW9c`).

## Step 2: Launch the Anyquery Server

Start the Anyquery server:

```bash
anyquery server
```

Because Metabase is a web-based tool, and Anyquery binds locally, you probably host Metabase on a remote server. Use a tool like [ngrok](https://ngrok.com/) to create a secure tunnel to your local server:

```bash
ngrok tcp 8070
```

Copy the forwarding URL (e.g., `tcp://0.tcp.ngrok.io:12345`) and use it as the hostname in the next step.

## Step 3: Connect Metabase to Anyquery

Go to the Metabase admin panel and add a new database connection:

1. Open Metabase in your browser and go to the database settings.
   `https://{your-metabase-url}/admin/databases/create`
2. Select MySQL as the database type.
3. Fill in the following details:
   - **Name**: A memorable name for the connection.
   - **Host**: The forwarding URL from ngrok (e.g., `0.tcp.ngrok.io`) or `127.0.0.1` if you are running Metabase locally.
   - **Port**: The port from ngrok (e.g., `12345`) or `8070` if you are running Metabase locally.
   - **Database name**: `main`.
   - **Username**: Set `root` unless you have set an [auth-file](https://anyquery.dev/docs/usage/mysql-server#adding-authentication).
   - **Password**: Leave it empty unless you have set an [auth-file](https://anyquery.dev/docs/usage/mysql-server#adding-authentication).
4. Click on the `Save` button to verify that the connection is successful.

![Metabase Connection](/images/docs/Ws1UhIKV.png)

## Step 4: Create Your First Visualization

Due to an unsolved bug in Anyquery, you cannot use the Metabase GUI to select tables. You need to create a model in Metabase with a native query. To do so, click on the `+ New` button on the top right and select `Model`, then `Use a native query`.

### Example Query

```sql
-- List all data from the Google Sheets spreadsheet
SELECT * FROM google_sheets_table;
```

Once you made your first query, you can create a new question and visualize the data. Click on the `+ New` button on the top right and select `Question`. You can now select the model you created and start building your visualization.

For example, here is a breakdown of some data from the spreadsheet:

![Metabase Visualization](/images/docs/Z2R9qeV3.png)

## Conclusion

You have successfully connected Metabase to Anyquery and visualized data from a Google Sheets spreadsheet. Now you can explore and visualize data from any source using Metabase. For more information, refer to the [Google Sheets plugin documentation](https://anyquery.dev/integrations/google_sheets).

Happy querying!
