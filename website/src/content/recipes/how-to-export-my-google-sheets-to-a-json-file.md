---
title: "How to export my Google Sheets to a JSON file?"
description: "Learn to export Google Sheets data to a JSON file using Anyquery. Follow steps to install the plugin, find your Spreadsheet ID, query data, and export it to JSON."
---

# How to Export Google Sheets to a JSON File

**Anyquery** is a powerful SQL query engine that allows you to run SQL queries on pretty much anything, including Google Sheets. This tutorial will guide you through the steps to export your Google Sheets data to a JSON file.

## Prerequisites

Before we begin, ensure you have the following:

- A working [installation of Anyquery](https://anyquery.dev/docs/#installation)
- A Google account
- The Google Sheets plugin installed

## Step 1: Install the Google Sheets Plugin

First, install the Google Sheets plugin if you haven't already:

```bash
anyquery install google_sheets
```

Follow the instructions to authenticate with Google. You will need to create a Google Cloud Project and enable the Google Sheets API. Refer to the [Google Sheets integration guide](https://anyquery.dev/integrations/google_sheets) for detailed setup steps.

## Step 2: Find the Spreadsheet ID

To export your Google Sheets data, you need the Spreadsheet ID. You can find it in the URL of your Google Sheets document. It is the string between `/d/` and `/edit` in the URL. For example, in the URL `https://docs.google.com/spreadsheets/d/1D_x7DNwbI9ZOSFjII6BvttPzdLJAymrQwQcOvnHzW9c/edit`, the Spreadsheet ID is `1D_x7DNwbI9ZOSFjII6BvttPzdLJAymrQwQcOvnHzW9c`.

## Step 3: Query Your Google Sheets Data

Once you have the Spreadsheet ID, you can query your Google Sheets data using Anyquery. First, test your connection by running the following command:

```bash
anyquery -q "SELECT * FROM google_sheets_table LIMIT 1"
```

Replace `google_sheets_table` with the name of your table.

## Step 4: Export to JSON

To export your Google Sheets data to a JSON file, run the following command:

```bash
anyquery -q "SELECT * FROM google_sheets_table" --json > data.json
```

This command will export all rows from your Google Sheets table to a file named `data.json`.

:::warning
Ensure you have sufficient permissions to write to the directory where you are executing the command.
:::

## Optional: Modifying Columns

You can also modify each column using SQL functions such as `upper`. For example, to convert the `name` column to uppercase, you can run:

```bash
anyquery -q "SELECT upper(name), age FROM google_sheets_table" --json > data.json
```

Refer to the [functions documentation](https://anyquery.dev/docs/reference/functions) for more information on the available functions.

## Conclusion

You have successfully exported your Google Sheets data to a JSON file using Anyquery. Now you can manipulate and use your data as needed. For more information on Anyquery and its capabilities, refer to the [official documentation](https://anyquery.dev/docs/).
