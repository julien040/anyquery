---
title: Metabase
description: Connect Metabase to Anyquery
---

<img src="/icons/metabase.svg" alt="Metabase" width="200"/>

Metabase is a powerful business intelligence tool that allows you to create and share data visualizations. You can connect Metabase to many data sources, including the MySQL server. Let's explore how to set up the connection.

## Prerequisites

Before starting, ensure you have the following:

- A working installation of Anyquery
- [Metabase](https://www.metabase.com/docs/latest/installation-and-operation/running-metabase-on-docker) running on Docker

## Step 1: Set up the connection

First, launch the Anyquery server:

```bash
anyquery server
```

Because Metabase is a web-based tool, and `anyquery` binds locally, you probably host metabase on a remote server. You can use a tool like [ngrok](https://ngrok.com/) to create a secure tunnel to your local server.

```bash
ngrok tcp 8070
```

Copy the forwarding URL (e.g., `tcp://0.tcp.ngrok.io:12345`) and use it as the hostname in the next step.

## Step 2: Connect Metabase

Go to the Metabase admin panel and add a new database connection:

1. Open Metabase in your browser and go to the database settings.
   `https://{your-metabase-url}/admin/databases/create`
2. Select MySQL as the database type.
3. Fill in the following details:
   - **Name**: A memorable name for the connection.
   - **Host**: The forwarding URL from ngrok (e.g., `0.tcp.ngrok.io`) or `127.0.0.1` if you are running Metabase locally.
   - **Port**: The port from ngrok (e.g., `12345`) or `8070` if you are running Metabase locally.
   - **Database name**: `main`.
   - **Username**: Set `root` unless you have set an [auth-file](/docs/usage/mysql-server#adding-authentication).
   - **Password**: Leave it empty unless you have set an [auth-file](/docs/usage/mysql-server#adding-authentication).
4. Click on the `Save` button to verify that the connection is successful.

![Metabase Connection](/images/docs/Ws1UhIKV.png)

## Running your first visualization

Due to an unsolved bug in anyquery, you cannot use the Metabase GUI to select tables. You need to create a model in Metabase with a native query. To do so, click on the `+ New` button on the top right and select `Model`, then `Use a native query`.

```sql
-- List my GitHub stars
SELECT * FROM github_my_stars;
```

Once you made your first query, you can create a new question and visualize the data. Click on the `+ New` button on the top right and select `Question`. You can now select the model you created and start building your visualization.

For example, here is the breakdown of my GitHub stars per language:

![Metabase Visualization](/images/docs/Z2R9qeV3.png)

## Conclusion

You have successfully connected Metabase to Anyquery. Now you can explore and visualize data from any source using Metabase.
