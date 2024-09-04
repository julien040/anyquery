---
title: "How to visualize a Google Sheets spreadsheet in Tableau?"
description: "Learn how to visualize Google Sheets data in Tableau using Anyquery. This guide covers setting up plugins, launching the server, and creating visualizations in Tableau."
---

# How to Visualize a Google Sheets Spreadsheet in Tableau

In this tutorial, we'll show you how to visualize data from a Google Sheets spreadsheet in Tableau using Anyquery. Anyquery is a versatile SQL query engine that can query data from various sources, including Google Sheets. We'll use Anyquery to access the Google Sheets data and then connect Tableau to visualize the data.

## Prerequisites

Before starting, ensure you have the following:

- A working installation of Anyquery (Refer to the [installation guide](https://anyquery.dev/docs/#installation))
- [Tableau Desktop](https://www.tableau.com/products/desktop/download) installed and activated
- Google Sheets plugin installed in Anyquery

## Step 1: Set Up Google Sheets Plugin

First, install the Google Sheets plugin in Anyquery. Follow the [Google Sheets plugin setup guide](https://anyquery.dev/integrations/google_sheets) to authenticate and configure the plugin.

```bash
anyquery install google_sheets
```

During the setup, you will be asked for:
- OAuth Client ID
- OAuth Client Secret
- Token (generated during the authentication process)
- Spreadsheet ID (found in the URL of your Google Sheets)

Complete the setup by following the instructions provided in the guide.

## Step 2: Launch Anyquery Server

Once the plugin is installed and configured, launch the Anyquery server to expose the data for Tableau.

```bash
anyquery server
```

## Step 3: Set Up Tableau Connection

1. Open Tableau Desktop and click on `MySQL` in the `Connect` pane (left side) under the section `To a Server`.

2. Fill in the following details:
   - **Server**: `127.0.0.1` (replace it with another IP if Anyquery binds to a different IP).
   - **Port**: `8070` (replace it with another port if Anyquery binds to a different port).
   - **Username**: Leave it empty unless you have set an [auth-file](https://anyquery.dev/docs/usage/mysql-server#adding-authentication).
   - **Password**: Leave it empty unless you have set an [auth-file](https://anyquery.dev/docs/usage/mysql-server#adding-authentication).
   - **Database**: `main`.

![Tableau Connection](/images/docs/vg6dOA3V.png)

3. Click on the `Sign In` button to verify that the connection is successful.

## Step 4: Visualize Data in Tableau

1. On the left sidebar, you can see the list of tables. Drag and drop the table corresponding to your Google Sheets data to the canvas to create a new worksheet.

2. Fill in the columns and rows to create your visualization. For example, you can create charts, graphs, and dashboards based on the data in your Google Sheets spreadsheet.

Here is an example of a breakdown of data from a Google Sheets spreadsheet visualized in Tableau:

![Tableau Visualization](/images/docs/tableau-github-stars.svg)

## Conclusion

You have successfully connected Tableau to Anyquery and visualized data from a Google Sheets spreadsheet. You can now create interactive dashboards and share them with your team. For more information, refer to [Tableau](https://www.tableau.com/products/desktop) and [Anyquery documentation](https://anyquery.dev/docs/).
