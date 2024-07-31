---
title: DBeaver
description: Connect DBeaver to Anyquery
---

![DBeaver](/images/docs/dbeaver.svg)

DBeaver is a free and open-source universal database tool for developers. You can use it to connect to almost any database, including the MySQL server. Let's explore how to set up the connection.

## Prerequisites

Before starting, ensure you have the following:

- A working installation of Anyquery
- DBeaver installed on your machine

## Starting MySQL Server

First, start the MySQL server of Anyquery:

```bash
anyquery server
```

## Connecting DBeaver

1. Open DBeaver and click on the plug icon in the top-left corner.
2. Select `MySQL` from the list of databases and click on `Next`.
3. Fill in the following details:
   - **Host**: `127.0.0.1` (replace it with another IP if Anyquery binds to a different IP).
   - **Port**: `8070` (replace it with another port if Anyquery binds to a different port).
   - **User**: Leave it empty unless you have set an [auth-file](/docs/usage/mysql-server#adding-authentication).
   - **Password**: Leave it empty unless you have set an [auth-file](/docs/usage/mysql-server#adding-authentication).
   - **Database**: `main`.
   - **Driver**: `MySQL (MariaDB)`.
4. Click on the `Test Connection` button to verify that the connection is successful.
5. If the test is successful, click on the `Finish` button to save the connection.

![New Connection](/images/docs/phTdzIiJ.png)

## Exploring and Querying Data

You can see the list of databases and tables on the left sidebar. Double-click any table to view its schema. Select the `data` tab to view its data. You can also run SQL queries by opening a new SQL editor tab and typing your query.

If the table supports updates, you can edit the data directly in DBeaver and click on the `Save` button to save the changes.

## Conclusion

You have successfully connected DBeaver to Anyquery. Now you can explore and query data from any source using DBeaver.
