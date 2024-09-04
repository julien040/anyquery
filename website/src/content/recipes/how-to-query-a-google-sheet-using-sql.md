---
title: "How to query a Google Sheet using SQL?"
description: "Learn to set up Anyquery, connect it to Google Sheets, and execute SQL queries on your data. Follow detailed steps for installation, authentication, and querying."
---

# How to Query a Google Sheet Using SQL

Anyquery is a versatile SQL query engine that allows you to run SQL queries on a variety of data sources, including Google Sheets. This tutorial will guide you through the steps to set up Anyquery, connect it to a Google Sheet, and query the data using SQL.

## Prerequisites

Before starting, ensure you have the following:

- A working installation of Anyquery. Follow the instructions here: [Anyquery Installation](https://anyquery.dev/docs/#installation).
- A Google account and access to Google Sheets.
- The Google Sheets plugin installed in Anyquery.

## Step 1: Install the Google Sheets Plugin

First, you need to install the Google Sheets plugin for Anyquery.

```bash
anyquery install google_sheets
```

## Step 2: Authenticate with Google

To connect Anyquery to your Google Sheets, you need to authenticate with Google. Follow these steps:

1. Go to the [Google Cloud Console](https://console.cloud.google.com/), create a new project, and navigate to the [APIs & Services console](https://console.cloud.google.com/apis/dashboard).

2. Click on Credentials.
   
   ![Google Cloud Console Credentials](https://cdn.jsdelivr.net/gh/julien040/anyquery@main/plugins/google_forms/images/identifier.png)

3. Click on Create Credentials, and select OAuth client ID.
   
   ![Create OAuth client ID](https://cdn.jsdelivr.net/gh/julien040/anyquery@main/plugins/google_forms/images/create.png)

4. If not done, configure the consent screen:
   - Select `External` and click on Create.
   - Fill the form with the required information:
     - Application name: Anyquery
     - User support email: Your email
     - Developer contact information: Your email
     - Leave the rest as it is.
   - Click on Save and Continue until you reach the Test Users section.
   - Add the Google account you will use to query the responses.
   - Click on Save and Continue.
   - Click on Back to Dashboard.

5. Go back to the Credentials tab and click on Create Credentials again. Select OAuth client ID, and select Web application.

6. Fill the form with the required information:
   - Leave the name as whatever you want.
   - Add the authorized redirect URIs: `https://integration.anyquery.dev/google-result`.
   - Add authorized JavaScript origins: `https://integration.anyquery.dev`.
   - Click on Create.
   
   ![OAuth Settings](https://cdn.jsdelivr.net/gh/julien040/anyquery@main/plugins/google_forms/images/form_oAuth.png)

7. Copy the `Client ID` and `Client Secret`. We will use them later.
   
   ![Copy Client ID and Secret](https://cdn.jsdelivr.net/gh/julien040/anyquery@main/plugins/google_forms/images/result.png)

8. Enable the Google Sheets API. To do so, go to the [Google Sheets API page](https://console.cloud.google.com/apis/library/sheets.googleapis.com) and click on Enable.

   ![Enable API](https://cdn.jsdelivr.net/gh/julien040/anyquery@main/plugins/google_sheets/images/enableAPI.png)

9. Go to [Google Sheets integration](https://integration.anyquery.dev/google-sheets).

10. Fill the form with the `Client ID` and `Client Secret` you copied and click on Submit.

    ![Google Sheets Integration](https://cdn.jsdelivr.net/gh/julien040/anyquery@main/plugins/google_forms/images/form_integration.png)

11. Select your Google account, skip the warning about the app not being verified, and copy the token, the client ID, and the client secret.

    ![Copy Token](https://cdn.jsdelivr.net/gh/julien040/anyquery@main/plugins/google_forms/images/token.png)

12. Go back to the terminal and fill in the form with the token, the client ID, and the client secret.
13. To find the spreadsheet ID, go to the spreadsheet edit page and copy the ID from the URL. For example, in `https://docs.google.com/spreadsheets/d/1D_x7DNwbI9ZOSFjII6BvttPzdLJAymrQwQcOvnHzW9c/edit?gid=1700564349#gid=1700564349`, the spreadsheet ID is `1D_x7DNwbI9ZOSFjII6BvttPzdLJAymrQwQcOvnHzW9c`.

## Step 3: Query Your Google Sheets

Now that you have set up the connection, you can query your Google Sheets using SQL. Here are some example queries:

```sql
-- List all the rows in the spreadsheet
SELECT * FROM google_sheets_spreadsheet;

-- List all the rows in the spreadsheet where the column "column_name" is equal to "value"
SELECT * FROM google_sheets_spreadsheet WHERE column_name = 'value';

-- Insert a new row in the spreadsheet
INSERT INTO google_sheets_spreadsheet (column_name1, column_name2) VALUES ('value1', 'value2');

-- Update the rows in the spreadsheet where the column "column_name" is equal to "value"
UPDATE google_sheets_spreadsheet SET column_name1 = 'value1', column_name2 = 'value2' WHERE column_name = 'value';

-- Delete the rows in the spreadsheet where the column "column_name" is equal to "value"
DELETE FROM google_sheets_spreadsheet WHERE column_name = 'value';
```

## Limitations

- The plugin only supports Google Sheets where the first row is the header row.
- The plugin does not support the `ALTER TABLE` command.
- Insertions, updates, and deletions are buffered for performance reasons.
- The plugin can handle 6000 insertions/modifications per minute and 60000 deletions per minute.
- Updating rows with formulas will replace the formula with the value.

For more details, refer to the [Google Sheets Plugin Documentation](https://anyquery.dev/integrations/google-sheets).

## Conclusion

You have successfully connected Anyquery to Google Sheets and performed SQL queries on your spreadsheet data. Now you can explore and manipulate your Google Sheets data with SQL queries.
