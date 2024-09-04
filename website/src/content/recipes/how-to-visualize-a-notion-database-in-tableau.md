---
title: "How to visualize a Notion database in Tableau?"
description: "Learn to connect Anyquery to Notion and visualize your data in Tableau, enabling creation of detailed reports and dashboards. Follow this step-by-step guide to get started."
---

# How to Visualize a Notion Database in Tableau

In this tutorial, you will learn how to connect Anyquery to Notion and visualize your data in Tableau. This can be useful for creating detailed reports and dashboards based on your Notion data.

## Prerequisites

Before you begin, ensure you have the following:

- A working installation of Anyquery. Follow the installation instructions here: [Anyquery Installation Guide](https://anyquery.dev/docs/#installation).
- A [Notion](https://www.notion.so/) account with a database you want to visualize.
- [Tableau Desktop](https://www.tableau.com/products/desktop/download) installed and activated.

## Step 1: Set Up the Notion Plugin in Anyquery

You'll need to install and configure the Notion Plugin for Anyquery:

1. **Install the Notion plugin**:

    ```bash
    anyquery install notion
    ```

2. **Create a Notion Integration and Get the Token**:
    - Go to [Notion's My Integrations page](https://www.notion.so/my-integrations).
    - Click on `+ New integration` and fill out the necessary information.
    - Copy the integration token.
    - Share the database with the integration by clicking on the `Share` button on the top right corner of your Notion database and selecting your integration.

3. **Provide the Integration Token and Database ID to Anyquery**:
    - When prompted during installation, provide the integration token and the Notion database ID.
    - The database ID can be found in the database URL. For example, in `https://www.notion.so/myworkspace/My-Database-1234567890abcdef1234567890abcdef`, the database ID is `1234567890abcdef1234567890abcdef`.

## Step 2: Launch the Anyquery Server

Start the Anyquery server to act as a MySQL server:

```bash
anyquery server
```

## Step 3: Expose Anyquery Server to the Internet

If you're running Tableau on a different machine, use a tool like [ngrok](https://ngrok.com/) to expose the Anyquery server to the internet:

```bash
ngrok tcp 8070
```

Copy the forwarding URL (e.g., `tcp://0.tcp.ngrok.io:12345`) and use it as the hostname in the next step.

## Step 4: Connect Tableau to Anyquery

1. **Open Tableau Desktop and click on `MySQL` in the `Connect` pane**:
2. **Fill in the following details**:
    - **Server**: Use `127.0.0.1` for local connections or the forwarding URL from ngrok (e.g., `0.tcp.ngrok.io`). 
    - **Port**: Use `8070` for local connections or the port from ngrok (e.g., `12345`).
    - **Username**: Set `root` unless you have set an [auth-file](https://anyquery.dev/docs/usage/mysql-server#adding-authentication).
    - **Password**: Leave it empty unless you have set an [auth-file](https://anyquery.dev/docs/usage/mysql-server#adding-authentication).
    - **Database**: `main`.

    ![Tableau Connection](/images/docs/vg6dOA3V.png)

3. **Click on the `Sign In` button** to verify that the connection is successful.

## Step 5: Visualize Your Notion Data in Tableau

1. On the left sidebar in Tableau, you will see the list of tables including your Notion database.
2. Drag and drop the Notion table to the canvas to create a new worksheet.
3. Use Tableau's visualization tools to create your dashboard. For example, you can drag columns to the Rows and Columns shelves, and use Marks to customize your visualizations.

## Example Visualization

Here is an example of a simple visualization that shows a breakdown of tasks by status from a Notion task database:

1. **Drag the `Status` column to the Rows shelf**.
2. **Drag the `Count` measure to the Columns shelf**.
3. **Use the `Marks` card to change the type of chart to a bar chart**.

![Tableau Visualization](/images/docs/tableau-github-stars.svg)

## Conclusion

You have successfully connected Tableau to Anyquery and visualized your Notion database. Now you can create interactive dashboards and share them with your team.

For more details and troubleshooting, refer to the following documentation:
- [Tableau Desktop Connection Help](https://www.tableau.com/support/drivers?edition=pro#mysql)
- [Anyquery Troubleshooting Guide](https://anyquery.dev/docs/usage/troubleshooting/)

Happy visualizing!
