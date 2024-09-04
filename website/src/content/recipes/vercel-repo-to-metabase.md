---
title: "Visualize the GitHub repositories of Vercel using Metabase"
description: "Learn how to connect Anyquery to the GitHub API and visualize the repositories of Vercel using Metabase"
---

Anyquery is a SQL query engine that allows you to run SQL queries on pretty much anything. In this recipe, we will connect Anyquery to the GitHub API and visualize the repositories of [Vercel](https://github.com/vercel) using Metabase.

## Prerequisites

Before starting, ensure you have the following:

- A working [installation of Anyquery](/docs/#installation)
- [Metabase](https://www.metabase.com/docs/latest/installation-and-operation/running-metabase-on-docker) running

## Step 1: Set up the connection

First, let's download the GitHub plugin for Anyquery. You'll need a GitHub token to access the GitHub API. You can create a token by following the [GitHub integration guide](/integrations/github).

```bash
anyquery install github
```

Next, launch the Anyquery server:

```bash
anyquery server
```

Because Metabase is a web-based tool, and `anyquery` binds locally, you probably host Metabase on a remote server. You can use a tool like [ngrok](https://ngrok.com/) to connect the local anyquery server to the internet.

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

Go back to the Metabase dashboard and create a new model with a native query. Click on the `+ New` button on the top right and select `Model`, then `Use a native query`.

![Metabase Model](/images/docs/GgCl8quP.png)

```sql
-- List the repositories of Vercel
SELECT * FROM github_repositories_from_user('vercel');
```

Click on the ▶️ button (or run `⌘ + enter`) to run the query. If it works, click on the `Save` button to save the model (you can name it `Vercel Repositories`).
Don't hesitate to modify the metadata of the columns to make the visualization more readable (for example, `updated_at` can be converted to a date, or add the suffix 'KB' to the `size` column).

![Metabase Model](/images/docs/rLpTGAGK.png)

Now, create a new question and visualize the data. Click on the `+ New` button on the top right and select `Question`. You can now select the model you created and start building your visualization.

Once you have created your questions, you can create a dashboard to visualize the data. Click on the `+ New` button on the top right and select `Dashboard`. You can now add the questions you created to the dashboard.

For example, here is a dashboard showing the repositories of Vercel:

![Metabase Dashboard](/images/docs/mfjW79d8.png)

## Conclusion

You have successfully connected Metabase to Anyquery and visualized the repositories of Vercel. Now you can explore and visualize data from any source using Metabase.
