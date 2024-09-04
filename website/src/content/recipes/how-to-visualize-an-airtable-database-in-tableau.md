---
title: "How to visualize an Airtable database in Tableau?"
description: "Learn to visualize Airtable data in Tableau via Anyquery. Install necessary plugins, configure connections, and create insightful visualizations with ease."
---

# How to Visualize an Airtable Database in Tableau

## Introduction

Anyquery lets you query and transform data from various sources including Airtable. Tableau is a powerful data visualization tool that can connect to many data sources, including MySQL. This tutorial will guide you on how to visualize data from an Airtable database in Tableau using Anyquery.

Before we start, a brief reminder: Anyquery is a SQL query engine that allows you to run SQL queries on various data sources. You must have Anyquery installed on your system. Refer to the [installation guide](https://anyquery.dev/docs/#installation) for details.

## Prerequisites

Before starting, ensure you have the following:

- A working installation of Anyquery
- [Tableau Desktop](https://www.tableau.com/products/desktop/download) installed and activated
- An Airtable account with the base and table you want to visualize

## Step 1: Install the Airtable Plugin in Anyquery

First, install the Airtable plugin for Anyquery:

```bash
anyquery install airtable
```

Next, follow the prompts to configure the plugin:

1. **Airtable API Key**: Go to the [Airtable tokens page](https://airtable.com/create/tokens) and create a new token with the required scopes (e.g., `data.records:read`, `schema.bases:read`). Copy the token.
2. **Airtable Base ID**: Open your Airtable base, and copy the string after `https://airtable.com/app` and before the `/` (e.g. `appWx9fD5JzAB4TIO`).
3. **Airtable Table Name**: Enter the name of the table you want to visualize.

Once configured, you can verify the connection by running:

```bash
anyquery -q "SELECT * FROM airtable_table LIMIT 1"
```

## Step 2: Start the Anyquery Server

Start the Anyquery server:

```bash
anyquery server
```

Because Tableau is a desktop application, you can connect directly to the local Anyquery server without additional steps.

## Step 3: Install the MySQL Connector for Tableau

Before connecting Tableau to Anyquery, ensure you have the MySQL connector installed. Follow the [installation instructions for MySQL connector](https://www.tableau.com/fr-fr/support/drivers?edition=pro#mysql).

## Step 4: Connect Tableau to Anyquery

1. Open Tableau Desktop.
2. In the `Connect` pane on the left side, select `MySQL` under the `To a Server` section.
3. Fill in the connection details:
   - **Server**: `127.0.0.1`
   - **Port**: `8070`
   - **Username**: Leave it empty unless you have set an [auth-file](https://anyquery.dev/docs/usage/mysql-server#adding-authentication).
   - **Password**: Leave it empty unless you have set an [auth-file](https://anyquery.dev/docs/usage/mysql-server#adding-authentication).
   - **Database**: `main`

![Tableau Connection](https://cdn.jsdelivr.net/gh/julien040/anyquery@main/plugins/tableau/images/tableau-connection.png)

4. Click the `Sign In` button to verify the connection.

## Step 5: Create Your First Visualization

1. On the left sidebar, you will see the list of tables. Drag and drop your Airtable table (e.g., `airtable_table`) to the canvas.
2. Fill in the columns and rows to create your visualization.

For example, here is a breakdown of tasks by status:

![Tableau Visualization](https://cdn.jsdelivr.net/gh/julien040/anyquery@main/plugins/tableau/images/tableau-visualization.png)

## Conclusion

You have successfully connected Tableau to Anyquery and visualized your Airtable database. Now you can create interactive dashboards and share them with your team.

For more details on using Anyquery with Tableau, refer to the [Tableau plugin documentation](https://anyquery.dev/integrations/tableau). For more details on using the Airtable plugin, refer to the [Airtable plugin documentation](https://anyquery.dev/integrations/airtable).
