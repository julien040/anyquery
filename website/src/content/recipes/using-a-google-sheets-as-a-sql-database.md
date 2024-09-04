---
title: "Using a Google Sheets as a SQL database"
description: "Learn to use Google Sheets as a SQL database with Anyquery. Install plugins, authenticate with Google, configure settings, and perform SQL operations seamlessly."
---

# Using Google Sheets as a SQL Database with Anyquery

Anyquery is a powerful SQL query engine that allows you to query data from various sources, including Google Sheets. In this tutorial, we will cover how to use a Google Sheets spreadsheet as a SQL database.

## Prerequisites

Before starting, ensure you have the following:

- A working [installation of Anyquery](https://anyquery.dev/docs/#installation)
- A Google account
- A Google Sheets spreadsheet

## Step 1: Install the Google Sheets Plugin

First, install the Google Sheets plugin for Anyquery. Run the following command:

```bash
anyquery install google_sheets
```

## Step 2: Set Up Google Cloud Console

You need to authenticate with Google to access your Google Sheets. Follow these steps:

1. Go to the [Google Cloud Console](https://console.cloud.google.com/).
2. Create a new project.
3. Navigate to the [APIs & Services Dashboard](https://console.cloud.google.com/apis/dashboard).
4. Click on **Credentials**.

### Create OAuth Client ID

1. Click on **Create Credentials**, and select **OAuth client ID**.
2. Configure the consent screen if prompted:
    - **Application type**: External
    - **Application name**: AnyQuery
    - Fill out the required fields and click **Save and Continue**.
    - Add the authorized redirect URI: `https://integration.anyquery.dev/google-result`
    - Add authorized JavaScript origins: `https://integration.anyquery.dev`
    - Click **Create**.
3. Copy the **Client ID** and **Client Secret**.

### Enable Google Sheets API

1. Go to the [Google Sheets API page](https://console.cloud.google.com/apis/library/sheets.googleapis.com).
2. Click **Enable**.

## Step 3: Authenticate with Google

1. Go to [Google Sheets integration](https://integration.anyquery.dev/google-sheets).
2. Fill in the **Client ID** and **Client Secret**, then click **Submit**.
3. Select your Google account and authorize the application.
4. Copy the token, client ID, and client secret provided.

## Step 4: Configure Anyquery

When prompted by Anyquery, provide the token, client ID, client secret, and the spreadsheet ID:

1. **Token**: Paste the token you copied.
2. **Client ID** and **Client Secret**: Paste these values.
3. **Spreadsheet ID**: Find the spreadsheet ID in the URL of your Google Sheets document. For example, in `https://docs.google.com/spreadsheets/d/1D_x7DNwbI9ZOSFjII6BvttPzdLJAymrQwQcOvnHzW9c/edit`, the spreadsheet ID is `1D_x7DNwbI9ZOSFjII6BvttPzdLJAymrQwQcOvnHzW9c`.

## Step 5: Querying Google Sheets

You can now use SQL to query your Google Sheets. Here are some examples:

### List All Rows

```sql
SELECT * FROM google_sheets_spreadsheet;
```

### Filter Rows by Column Value

```sql
SELECT * FROM google_sheets_spreadsheet WHERE column_name = 'value';
```

### Insert a New Row

```sql
INSERT INTO google_sheets_spreadsheet (column_name1, column_name2) VALUES ('value1', 'value2');
```

### Update Rows

```sql
UPDATE google_sheets_spreadsheet SET column_name1 = 'new_value' WHERE column_name = 'value';
```

### Delete Rows

```sql
DELETE FROM google_sheets_spreadsheet WHERE column_name = 'value';
```

## Limitations

- **Header Row**: The first row of the spreadsheet must be the header row.
- **Buffering**: Insertions and updates are buffered to avoid excessive API calls.
- **Rate Limits**: Google Sheets API has rate limits. You can insert or modify up to 6000 rows per minute and delete up to 60000 rows per minute.

## Conclusion

You have successfully set up Anyquery to use Google Sheets as a SQL database. Now you can perform various SQL operations on your Google Sheets data efficiently. Happy querying!

For more detailed information, you can refer to the [Google Sheets plugin documentation](https://anyquery.dev/integrations/google_sheets).
