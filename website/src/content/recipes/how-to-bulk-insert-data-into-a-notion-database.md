---
title: "How to bulk insert data into a Notion database?"
description: "Learn to bulk insert data into a Notion database using Anyquery. This step-by-step guide covers data preparation, reading from CSV, SQL queries, and verification."
---

# How to Bulk Insert Data into a Notion Database

Anyquery is a SQL query engine that allows you to query and manipulate data from various sources, including Notion databases. One of the common tasks is to bulk insert data into a Notion database. This guide will show you how to do it step-by-step.

## Prerequisites

Before we start, ensure you have the following:

- A working [installation of Anyquery](https://anyquery.dev/docs/#installation)
- The Notion plugin installed: `anyquery install notion`
- A Notion integration set up and authorized to access your database. Follow the [Notion plugin setup guide](https://anyquery.dev/integrations/notion) to get your API key and database ID.

## Step 1: Prepare Your Data

First, you need the data you want to insert into the Notion database. This data can come from a CSV file, another database, or any other source. For this example, we'll assume you have a CSV file named `data.csv`.

### Sample Data (`data.csv`)

```csv
Name,Email,Age
Alice,alice@example.com,28
Bob,bob@example.com,34
Charlie,charlie@example.com,22
```

## Step 2: Create a Notion Database

Ensure you have a Notion database set up with the appropriate schema. For this example, the database should have columns `Name`, `Email`, and `Age`.

## Step 3: Read Data from the CSV File

You can use Anyquery to read the data from the CSV file using the `read_csv` function. 

```sql
SELECT * FROM read_csv('path/to/data.csv', header=true);
```

## Step 4: Bulk Insert Data into Notion Database

Now, let's bulk insert the data into the Notion database. You should have your `notion_database` table configured properly. Here's how you can achieve this:

### SQL Query for Bulk Insert

```sql
INSERT INTO notion_database (Name, Email, Age)
SELECT Name, Email, Age FROM read_csv('path/to/data.csv', header=true);
```

### Running the Command

You can run the above query using Anyquery in the shell mode or as a one-off command. Here are both methods:

#### Shell Mode

1. Open the Anyquery shell:
   ```bash
   anyquery
   ```
2. Run the initialization query to ensure the plugin and database are set up:
   ```sql
   .init "path/to/your/init.sql"
   ```
3. Execute the bulk insert query:
   ```sql
   INSERT INTO notion_database (Name, Email, Age)
   SELECT Name, Email, Age FROM read_csv('path/to/data.csv', header=true);
   ```

#### One-Off Command

```bash
anyquery -q "INSERT INTO notion_database (Name, Email, Age) SELECT Name, Email, Age FROM read_csv('path/to/data.csv', header=true);"
```

## Step 5: Verify the Data

After running the insert command, you should verify the data to ensure it has been inserted correctly. You can query the Notion database to check the inserted records.

### SQL Query to Verify Data

```sql
SELECT * FROM notion_database;
```

## Conclusion

You have successfully bulk inserted data into a Notion database using Anyquery. This method can be applied to any data source that Anyquery supports, making it a powerful tool for data manipulation and integration.

For more details on the Anyquery Notion plugin, please refer to the [Notion plugin documentation](https://anyquery.dev/integrations/notion).

Happy querying!
