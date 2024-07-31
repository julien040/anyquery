---
title: Looker Studio (Data Studio)
description: Connect Looker Studio to Anyquery
---

![Looker Studio](/images/docs/looker.svg)

Looker Studio is a powerful business intelligence tool that allows you to create and share data visualizations. You can connect Looker Studio to many data sources, including the MySQL server. Let's explore how to set up the connection.

## Prerequisites

Before starting, ensure you have the following:

- A working installation of Anyquery
- A Google account to access Looker Studio

## Step 1: Set up the connection

First, launch the Anyquery server:

```bash
anyquery server
```

Because Looker Studio is a web-based tool, and `anyquery` binds locally, you need to expose the server to the internet. You can use a tool like [ngrok](https://ngrok.com/) to create a secure tunnel to your local server.

```bash
ngrok tcp 8070
```

Copy the forwarding URL (e.g., `tcp://0.tcp.ngrok.io:12345`) and use it as the hostname in the next step.

## Step 2: Connect Looker Studio

1. Open [Looker Studio](https://lookerstudio.google.com/u/0/navigation/reporting) in your browser.
2. Click on the `+` icon (empty report) to create a new report.
3. In the search bar, type `MySQL` and select the `MySQL` connection.
4. Authorize Looker Studio to access your data.
5. Fill in the following details:
   - **Host**: The forwarding URL from ngrok (e.g., `0.tcp.ngrok.io`).
   - **Port**: The port from ngrok (e.g., `12345`).
   - **Database**: `main`.
   - **Username**: Set `root` unless you have set an [auth-file](/docs/usage/mysql-server#adding-authentication).
   - **Password**: Leave it empty unless you have set an [auth-file](/docs/usage/mysql-server#adding-authentication).
6. Selecting a table does not work in Looker Studio. You can only run SQL queries by clicking on the "Personalized query" tab and typing your query. Often, you simply want to `SELECT * FROM table_name`.
7. Click on the `Authenticate` button to verify that the connection is successful.

:::caution
Looker Studio does not handle all column names well. If you have a column name with a space or `É` for example, you might not be able to pick it up as a dimension or measure. You can rename the column in the query to work around this issue. For example, `SELECT \`État\` AS \`etat\` FROM table_name`.
:::

## Example of a dashboard

I have a made a Looker Studio dashboard of the backlog of plugins for Anyquery. I have a Notion board where I track the progress of each plugin. Using anyquery, I was able to query this board and show the progress in Looker Studio.

![Looker Studio Dashboard](/images/docs/oE0a8dtb.png)

## Conclusion

You have successfully connected Looker Studio to Anyquery. Now you can explore and visualize data from any source using Looker Studio.
