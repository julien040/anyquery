---
title: "How to visualize an Airtable database in Metabase?"
description: "Learn to connect Anyquery to an Airtable database and visualize it in Metabase. Follow steps to install plugins, configure connections, and create visualizations."
---

# How to Visualize an Airtable Database in Metabase

In this tutorial, we'll learn how to connect Anyquery to an Airtable database and visualize it using Metabase. Anyquery is a SQL query engine that enables you to query data from various sources, including Airtable. Metabase is a powerful business intelligence tool that allows you to create and share data visualizations.

## Prerequisites

Before starting, ensure you have the following:

- A working installation of Anyquery
- An Airtable account with a database (you must create the Airtable database before using it in Anyquery)
- Metabase running on Docker (instructions available on [Metabase's website](https://www.metabase.com/docs/latest/installation-and-operation/running-metabase-on-docker))

For installing Anyquery, refer to the [installation documentation](https://anyquery.dev/docs/#installation).

## Step 1: Set up Anyquery with Airtable

1. **Install Anyquery and Airtable Plugin**:
    Run the following command to install the Airtable plugin for Anyquery:
    ```bash
    anyquery install airtable
    ```

2. **Authenticate with Airtable**:
    Follow these steps to get your Airtable API key and set up Anyquery:
    - Go to the [Airtable API page](https://airtable.com/api) and sign in.
    - Select your base (database) from the list.
    - Copy the API key from the top right corner of the page.
    - Use the API key and the base ID to configure the Airtable plugin in Anyquery. The base ID can be found in the URL of the base (e.g., in `https://airtable.com/appWx9fD5JzAB4TIO/tblnTJJsUb8f7QjiM/viwDj8WIME?blocks=hide`, the base ID is `appWx9fD5JzAB4TIO`).

    ```bash
    anyquery profile new default airtable my_airtable
    ```

    Follow the prompts to input your API key and base ID.

3. **Verify Connection**:
    Run the following command to ensure the connection is working:
    ```bash
    anyquery -q "SELECT * FROM my_airtable_airtable_table LIMIT 1"
    ```

## Step 2: Launch Anyquery Server

Start the Anyquery server to allow remote connections:

```bash
anyquery server
```

Because Metabase is a web-based tool and Anyquery binds locally, you need to expose the server to the internet. Use a tool like [ngrok](https://ngrok.com/) to create a secure tunnel:

```bash
ngrok tcp 8070
```

Copy the forwarding URL (e.g., `tcp://0.tcp.ngrok.io:12345`) and use it as the hostname in the next step.

## Step 3: Connect Metabase to Anyquery

1. **Open Metabase Admin Panel**:
    - Open Metabase in your browser and navigate to the database settings: `https://{your-metabase-url}/admin/databases/create`

2. **Add a New Database Connection**:
    - Select MySQL as the database type.
    - Fill in the following details:
        - **Name**: A memorable name for the connection (e.g., "Airtable Database").
        - **Host**: The forwarding URL from ngrok (e.g., `0.tcp.ngrok.io`) or `127.0.0.1` if you are running Metabase locally.
        - **Port**: The port from ngrok (e.g., `12345`) or `8070` if you are running Metabase locally.
        - **Database name**: `main`.
        - **Username**: Set `root` unless you have set an [auth-file](https://anyquery.dev/docs/usage/mysql-server#adding-authentication).
        - **Password**: Leave it empty unless you have set an [auth-file](https://anyquery.dev/docs/usage/mysql-server#adding-authentication).

    ![Metabase Connection](/images/docs/Ws1UhIKV.png)

3. **Save and Test Connection**:
    - Click on the `Save` button to verify that the connection is successful.

## Step 4: Create Visualizations in Metabase

1. **Create a New Model with a Native Query**:
   - Click on the `+ New` button on the top right and select `Model`, then `Use a native query`.
   - Write a SQL query to select data from your Airtable database:
     ```sql
     SELECT * FROM my_airtable_airtable_table;
     ```
   - Click on the ▶️ button (or run `⌘ + enter`) to run the query.
   - Save the model (you can name it "Airtable Data").

2. **Create a New Question**:
   - Click on the `+ New` button on the top right and select `Question`.
   - Select the model you created and start building your visualization.
   - For example, you can create a bar chart to visualize the data.

3. **Create a Dashboard**:
   - Click on the `+ New` button on the top right and select `Dashboard`.
   - Add the questions you created to the dashboard.
   - Arrange the visualizations as needed.

For example, here is a dashboard showing data from an Airtable database:

![Metabase Dashboard](/images/docs/mfjW79d8.png)

## Conclusion

You have successfully connected Metabase to Anyquery and visualized an Airtable database. Now you can explore and visualize data from Airtable using Metabase. For more information on using Metabase, refer to the [official Metabase documentation](https://www.metabase.com/docs/latest/installation-and-operation/running-metabase-on-docker).

For troubleshooting and more details, refer to [Anyquery troubleshooting](https://anyquery.dev/docs/usage/troubleshooting/) and [Metabase documentation](https://www.metabase.com/docs/latest/installation-and-operation/running-metabase-on-docker).
