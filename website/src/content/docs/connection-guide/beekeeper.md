---
title: Beekeeper
description: Connect Beekeeper to Anyquery
---

<img src="/icons/beekeeper.svg" alt="Beekeeper" width="200"/>

## Prerequisites

Before you begin, ensure that you have the following:

- A working installation of Anyquery
- Beekeeper installed on your machine

## Step 1: Set up the connection

First, open Beekeeper and click on the `+ New Connection` button. Then, fill in the following details:

- Select `MySQL` for the dropdown `Select a connection type`.
- Host: `127.0.0.1` (replace it with another IP if any query binds to a different IP).
- Port: `8070` (replace it with another port if any query binds to a different port).
- User: Leave it empty unless you have set an [auth-file](/docs/usage/mysql-server#adding-authentication).
- Password: Leave it empty unless you have set an [auth-file](/docs/usage/mysql-server#adding-authentication).
- Default database: `main`.
- Name: `<Whatever you want>` (e.g. `Anyquery`).

![New Connection](/images/docs/ePlXiEg3.png)

## Step 2: Launch the MySQL server on Anyquery

Launch the Anyquery server in a terminal:

```bash
anyquery server
```

Back to beekeeper, click on the `Test` button next to `Connect` to ensure that the connection is successful. If successful, click on `Save` to save the connection.

## Step 3: Explore and query the data

Click on the connection you just created in the sidebar to view its data. You can double-click on any table to view its data if it doesn't require a parameter. You can also run SQL queries by opening a new tab and typing your query.

Row editing is not supported yet, but it's on the roadmap.
