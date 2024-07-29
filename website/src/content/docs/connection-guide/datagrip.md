---
title: DataGrip
description: Connect DataGrip to Anyquery
---

![DataGrip](https://cdn.svgporn.com/logos/datagrip.svg)

Let's connect DataGrip to Anyquery so that you can explore its data with ease.

## Prerequisites

Before you begin, ensure that you have the following:

- A working installation of Anyquery
- DataGrip installed on your machine

## Step 1: Set up the connection

First, open a DataGrip project and click on the `+` icon to add a new data source.
Select `Data Source` and choose `MySQL` as the database type. Then, fill in the following details:

- Host: `127.0.0.1`
- Port: `8070`
- User: Leave it empty unless you have set an [auth-file](/docs/usage/mysql-server#adding-authentication)
- Password: Leave it empty unless you have set an [auth-file](/docs/usage/mysql-server#adding-authentication)
- Database: `main`

![Data Source](/images/docs/O08kLX6m.png)

Launch the Anyquery server on a second terminal:

```bash
anyquery server
```

Then, click on the `Test Connection` button to ensure that the connection is successful. If successful, click on `OK` to save the connection.

## Step 2: Explore and query the data

You can double-click on any table in the sidebar to view its data. Note that it can take a long time on slow plugins. You can edit data like a spreadsheet on plugins that supports it (e.g. Google Sheets, Airtable, etc.).

You can also run SQL queries by clicking on the `Console` tab and typing your query.

Feel free to check the [official DataGrip documentation](https://www.jetbrains.com/help/datagrip/getting-started.html) for more information.

## Conclusion

Congratulations! You have successfully connected DataGrip to Anyquery. 🎉
