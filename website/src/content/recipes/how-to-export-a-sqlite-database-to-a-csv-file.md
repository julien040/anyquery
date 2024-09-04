---
title: "How to export a SQLite database to a CSV file?"
description: "Learn to export data from a SQLite database to a CSV file using Anyquery. This guide covers installation, connecting to the database, running export queries, and verifying output."
---

# How to Export a SQLite Database to a CSV File

Exporting data from a SQLite database to a CSV file can be very useful for data analysis, reporting, and sharing. With the help of Anyquery, you can easily achieve this by running a simple SQL query. This tutorial will guide you through the steps to export a SQLite database to a CSV file.

## Prerequisites

Before starting, ensure you have the following:
- A working installation of Anyquery. Follow the [installation guide](https://anyquery.dev/docs/#installation) if you haven't installed it yet.
- A SQLite database file that you want to export.

## Step-by-Step Guide

### Step 1: Launch Anyquery

Open your terminal and launch Anyquery.

```bash
anyquery
```

### Step 2: Connect to Your SQLite Database

Use the `-d` flag to specify the path to your SQLite database file. For example, if your database is named `mydatabase.db`, use the following command:

```bash
anyquery -d mydatabase.db
```

### Step 3: Run the Export Query

Use the `-q` flag to specify your SQL query and the `--csv` flag to specify the output format. Redirect the output to a CSV file using the `>` operator.

For example, to export all data from a table named `my_table` to a CSV file named `output.csv`, run the following command:

```bash
anyquery -q "SELECT * FROM my_table" --csv > output.csv
```

### Step 4: Verify the Output

Open the `output.csv` file to ensure that the data has been correctly exported. You can use any text editor or spreadsheet software to view the CSV file.

## Additional Tips

- **Filtering Data**: You can add WHERE clauses to filter the data being exported. For example:
  ```bash
  anyquery -q "SELECT * FROM my_table WHERE column_name = 'value'" --csv > output.csv
  ```

- **Selecting Specific Columns**: If you only need specific columns, adjust your SELECT statement accordingly:
  ```bash
  anyquery -q "SELECT column1, column2 FROM my_table" --csv > output.csv
  ```

- **Exporting Multiple Tables**: If you need to export multiple tables, run separate commands for each table.

- **Using Functions**: You can use SQL functions to modify the data before exporting. For example, to convert a column to uppercase:
  ```bash
  anyquery -q "SELECT UPPER(column_name) FROM my_table" --csv > output.csv
  ```

For more information on available functions, refer to the [functions documentation](https://anyquery.dev/docs/reference/functions).

## Conclusion

You have successfully exported a SQLite database to a CSV file using Anyquery. This method is straightforward and flexible, allowing you to filter and format your data as needed. Now you can easily share your data or use it for further analysis in tools like Excel or Google Sheets.
