# Google Sheets plugin

This plugin allows you to use a Google Sheets spreadsheet as a SQL database.
You can insert/update/delete/select data from the spreadsheet using SQL queries.

> **Limitation:**
>
> The plugin only supports that the first row of the spreadsheet is the header row.

## Setup

Install the plugin with:

```bash
anyquery install google_sheets
```

Then, you need to authenticate with Google. Go to the [Google Cloud Console](https://console.cloud.google.com/), create a new project, and go to the [APIs & Services console](https://console.cloud.google.com/apis/dashboard).

1. Click on Credentials
    ![alt text](https://cdn.jsdelivr.net/gh/julien040/anyquery@main/plugins/google_forms/images/identifier.png)
2. Click on Create Credentials, and select OAuth client ID
    ![alt text](https://cdn.jsdelivr.net/gh/julien040/anyquery@main/plugins/google_forms/images/create.png)
3. If not done, configure the consent screen
    ![alt text](https://cdn.jsdelivr.net/gh/julien040/anyquery@main/plugins/google_forms/images/consentScreen.png)
    1. Select `External` and click on Create
    2. And fill the form with the required information
        - Application name: AnyQuery
        - User support email: Your email
        - Developer contact information: Your email
        - Leave the rest as it is

        ![alt text](https://cdn.jsdelivr.net/gh/julien040/anyquery@main/plugins/google_forms/images/consentFilled.png)
    3. Click on Save and Continue
    4. Click on Save and Continue and leave Scopes as it is
    5. On test users, add the Google account you will use to query the responses
    6. Click on Save and Continue
    7. Click on Back to Dashboard
4. Go back to the Credentials tab and click on Create Credentials
    ![alt text](https://cdn.jsdelivr.net/gh/julien040/anyquery@main/plugins/google_forms/images/createCredentials.png)
5. Select OAuth client ID, and select Web application
6. Fill the form with the required information
    - Leave the name as whatever you want
    - Add the authorized redirect URIs: `https://integration.anyquery.dev/google-result`
    - Add authorized JavaScript origins: `https://integration.anyquery.dev`
    - Click on Create
    ![alt text](https://cdn.jsdelivr.net/gh/julien040/anyquery@main/plugins/google_forms/images/form_oAuth.png)
7. Copy the `Client ID` and `Client Secret`. We will use them later
    ![alt text](https://cdn.jsdelivr.net/gh/julien040/anyquery@main/plugins/google_forms/images/result.png)
8. Enable the Google Sheets API. To do so, go to the [Google Sheets API page](https://console.cloud.google.com/apis/library/sheets.googleapis.com) and click on Enable
9. ![alt text](https://cdn.jsdelivr.net/gh/julien040/anyquery@main/plugins/google_sheets/images/enableAPI.png)
10. Go to [Google Sheets integration](https://integration.anyquery.dev/google-sheets)
11. Fill the form with the `Client ID` and `Client Secret` you copied and click on Submit
    ![alt text](https://cdn.jsdelivr.net/gh/julien040/anyquery@main/plugins/google_forms/images/form_integration.png)
12. Select your Google account, skip the warning about the app not being verified, and
13. Copy the token, the client ID, and the client secret
    ![alt text](https://cdn.jsdelivr.net/gh/julien040/anyquery@main/plugins/google_forms/images/token.png)
14. Go back to the terminal and fill in the form with the token, the client ID, and the client secret.
15. To find the form ID, go to the spreadsheet edit page and copy the ID from the URL
   In `https://docs.google.com/spreadsheets/d/1D_x7DNwbI9ZOSFjII6BvttPzdLJAymrQwQcOvnHzW9c/edit?gid=1700564349#gid=1700564349`, the form ID is `1D_x7DNwbI9ZOSFjII6BvttPzdLJAymrQwQcOvnHzW9c`

When `anyquery` finishes the installation, you will be asked to provide the token, the client ID, the client secret, and the spreadsheet ID. Once you have provided the information, the plugin will be ready to use.

## Usage

Cell values are automatically converted to the appropriate type by Google Sheets when inserting/modifying. Formatted numbers are passed as floats to SQLite, without the formatting. When a cell is a formula, the value is the result of the formula. However, when modifying a cell with a formula, the plugin will replace the formula with the value (this is a limitation that I hope I will be able to fix in the future).

```sql
-- List all the rows in the spreadsheet
SELECT * FROM google_sheets_spreadsheet
-- List all the rows in the spreadsheet where the column "column_name" is equal to "value"
SELECT * FROM google_sheets_spreadsheet WHERE column_name = 'value'
-- Insert a new row in the spreadsheet
INSERT INTO google_sheets_spreadsheet (column_name1, column_name2) VALUES ('value1', 'value2')
-- Update the rows in the spreadsheet where the column "column_name" is equal to "value"
UPDATE google_sheets_spreadsheet SET column_name1 = 'value1', column_name2 = 'value2' WHERE column_name = 'value'
-- Delete the rows in the spreadsheet where the column "column_name" is equal to "value"
DELETE FROM google_sheets_spreadsheet WHERE column_name = 'value'
```

## Schema

Each table has a rowIndex. The other columns are the header row of the spreadsheet. The column type is inferred from the data in the spreadsheet.
However, sometimes the plugin can't infer the type, and it will default to `REAL`. Due to the nature of SQLite, this is not an issue. Indeed, SQLite is a dynamic type system, and you can store any type in any column. Types are just affinities.

| Column index | Column name | type    |
| ------------ | ----------- | ------- |
| 0            | rowIndex    | INTEGER |

## Limitations

*I hope I'll be able to address these limitations in the future.*

- The plugin only supports that the first row of the spreadsheet is the header row. You cannot yet have a table at an arbitrary row.
- The plugin does not support the `ALTER TABLE` command. You have to create the spreadsheet with the correct columns from the start.
- Insertion and update does not work well with smart chips. The plugin will try to insert the smart chip as a string, which will not work.
- Delete can be done 1000 rows at a time. Trying to delete more than 1000 rows will probably result in an error due to the nature of sucessive delete requests.
- Insert/modification/deletion of rows are buffered. This means that the changes are not immediately visible in the spreadsheet. The changes are applied when the buffer is full or when a SELECT query is run. This is to avoid making too many requests to the Google Sheets API (buffer length is 100 for insertion/modification and 1000 for deletion).
- It therefore means that you can do 6000 insertions/modifications per minute and 60000 deletions per minute. You can read 120 000 rows per minute.
- Updating rows with formulas will replace the formula with the value.
