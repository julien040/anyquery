---
title: TablePlus
description: Connect TablePlus to Anyquery
---

<img src="/icons/tablePlus.png" alt="TablePlus" width="180"/>

TablePlus is a user-friendly GUI tool for managing relational databases. You can use it to connect to any data source, including the MySQL server. Let's explore how to set up the connection.

## Prerequisites

Before starting, ensure you have the following:

- A working installation of Anyquery
- TablePlus installed on your machine

## Starting MySQL Server

First, start the MySQL server of anyquery:

```bash
anyquery server
```

## Connecting TablePlus

1. Open TablePlus, and click on the plug icon in the navigation bar.
2. Click the `New` button to create a new connection.
3. Select `MySQL` from the list of databases.
4. Fill in the form
   1. **Name**: Enter any desired name for the connection.
   2. **Host**: `127.0.0.1` (replace it with another IP if anyquery binds to a different IP).
   3. **Port**: `8070` (replace it with another port if anyquery binds to a different port).
   4. **User**: Leave it empty unless you have set an [auth-file](/docs/usage/mysql-server#adding-authentication).
   5. **Password**: Leave it empty unless you have set an [auth-file](/docs/usage/mysql-server#adding-authentication).
   6. **Database**: `main`.
5. Click on the `Test` button to verify that the connection is successful.
6. If the test is successful, click on the `Connect` button to save the connection and establish it.
![New Connection](/images/docs/J028Akph.png)

## Exploring and Querying Data

On the left sidebar, you can see the list of databases and tables. Double-click any table to view its data if it doesn't require a parameter. You can also run SQL queries by opening a new tab and typing your query.

For tables that support update, you can edit the data directly in TablePlus and hit `Cmd + S` to save the changes.

## Conclusion

You have successfully connected TablePlus to Anyquery. Now you can explore and query data from any source using TablePlus.
