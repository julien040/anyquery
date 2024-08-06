---
title: Redash
description: Connect Redash to Anyquery
---

<img src="/icons/redash.svg" alt="Redash" width="200"/>

Redash is a powerful business intelligence tool that allows you to create and share data visualizations. You can connect Redash to many data sources, including the MySQL server. Let's explore how to set up the connection.

## Prerequisites

Before starting, ensure you have the following:

- A working installation of Anyquery
- A Redash account

## Step 1: Set up the connection

First, launch the Anyquery server:

```bash
anyquery server
```

Because Redash is a web-based tool, and `anyquery` binds locally, you need to expose the server to the internet. You can use a tool like [ngrok](https://ngrok.com/) to create a secure tunnel to your local server.

```bash
ngrok tcp 8070
```

Copy the forwarding URL (e.g., `tcp://0.tcp.ngrok.io:12345`) and use it as the hostname in the next step.

## Step 2: Connect Redash

1. Open your redash instance in your browser.
2. Go to `{your-redash-url}/data_sources/new`.
3. Select MySQL as the data source.
4. Fill in the following details:
   - **Name**: A memorable name for the connection.
   - **Host**: The forwarding URL from ngrok (e.g., `0.tcp.ngrok.io`).
   - **Port**: The port from ngrok (e.g., `12345`).
   - **Database Name**: `main`.
   - **Username**: Set `root` unless you have set an [auth-file](/docs/usage/mysql-server#adding-authentication).
   - **Password**: Leave it empty unless you have set an [auth-file](/docs/usage/mysql-server#adding-authentication).
  ![Redash Connection](/images/docs/P6MjoIjz.png)

5. Click on the `Create` button to save the connection. Then click on `Test Connection` to verify that the connection is successful.

## Creating your first visualization

1. Go to the Redash dashboard and click on the `+ New Query` button.
2. Select the data source you just created.
3. Write your query in the SQL editor. Often, you simply want to `SELECT * FROM table_name`. Then, click on the `Execute` button. Finally, click on the `Publish` button to save the query.
4. Go to the dashboard and click on the `+ New Dashboard` button. Input a name for the dashboard.
5. Add a new widget to the dashboard and select the query you just created.

## Conclusion

You have successfully connected Redash to Anyquery. Now you can explore and visualize data from any source using Redash.
