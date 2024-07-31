---
title: MySQL Workbench
description: Connect MySQL Workbench to Anyquery
---
<img src="/icons/mysql-workbench.png" alt="MySQL Workbench" width="180"/>

MySQL Workbench is the de facto standard for MySQL database management. You can use it to explore and query data from `anyquery`. Let's explore how to set up the connection.

## Prerequisites

Before starting, ensure you have the following:

- A working installation of Anyquery
- MySQL Workbench installed on your machine

## Starting MySQL Server

First, start the MySQL server of Anyquery:

```bash
anyquery server
```

## Connecting MySQL Workbench

1. Open MySQL Workbench and click on the `+` icon (next to `MySQL Connections`) to add a new connection.
2. Fill in the following details:
   - **Connection Name**: Enter any desired name for the connection.
   - **Connection Method**: Select `Standard (TCP/IP)`.
   - **Hostname**: `127.0.0.1` (replace it with another IP if Anyquery binds to a different IP).
   - **Port**: `8070` (replace it with another port if Anyquery binds to a different port).
   - **Username**: Leave it as is unless you have set an [auth-file](/docs/usage/mysql-server#adding-authentication).
   - **Password**: Leave it empty unless you have set an [auth-file](/docs/usage/mysql-server#adding-authentication).
   - **Default Schema**: `main`.
3. Click on the `Test Connection` button to verify that the connection is successful.
4. If the test is successful, click on the `OK` button to save the connection.

![New Connection](/images/docs/Id89086t.png)

## Exploring and Querying Data

Double-click on the connection to establish a connection to the server. You can see the list of databases and tables on the left sidebar. Right-click on any table to open the table inspector or select the first 1000 rows. You can also run SQL queries by clicking on the `Query` tab and typing your query.

:::note
You might stumble upon a lot of warnings and errors when trying to inspect the table. Because anyquery is not a MySQL server, MySQL Workbench might expect some features that are not supported by anyquery. It's an ongoing effort to make anyquery compatible with more MySQL clients.
As a workaround, you can just click on `OK` and reduce the number of features that MySQL Workbench tries to use.
:::

## Conclusion

You have successfully connected MySQL Workbench to Anyquery. Now you can explore and query data from any source using MySQL Workbench.
