---
title: "How to convert a CSV file to a JSON file?"
description: "Learn to convert CSV to JSON using Anyquery. Follow steps to query CSV files, export results to JSON, modify columns, and handle errors effectively."
---

# How to Convert a CSV File to a JSON File

Anyquery is a SQL query engine that enables you to execute SQL queries on various data sources, including CSV files. Additionally, it can export query results to different formats like JSON, which allows for easy conversion of a CSV file to a JSON file.

## Prerequisites

Before you begin, ensure the following:

- You have Anyquery installed on your machine. For installation instructions, refer to the [installation guide](https://anyquery.dev/docs/#installation).

## Step-by-Step Guide

### Step 1: Open Anyquery Shell

First, open the Anyquery shell by running:

```bash
anyquery
```

### Step 2: Query the CSV File

To convert a CSV file to JSON, use the `read_csv` function. This function takes the path to the CSV file as an argument and reads the content of the file. The following example assumes that your CSV file is located at `path/to/file.csv`.

```sql
SELECT * FROM read_csv('path/to/file.csv');
```

### Step 3: Export the Result to JSON

You can export the result of your query to JSON format using the `--json` flag or by setting the output format within the shell. Here are both methods:

**Method 1: Using the `--json` flag**

```bash
anyquery -q "SELECT * FROM read_csv('path/to/file.csv')" --json > file.json
```

**Method 2: Setting the output format within the shell**

1. Set the output format to JSON:
    ```sql
    .json
    ```

2. Run your query:
    ```sql
    SELECT * FROM read_csv('path/to/file.csv');
    ```

3. Redirect the output to a JSON file:
    ```sql
    .output file.json
    ```

### Step 4: Modify Columns (Optional)

You can also modify each column using functions such as `upper`, `lower`, `CAST`, etc. For example, the following query converts the `name` column to uppercase before exporting to JSON:

```bash
anyquery -q "SELECT upper(name) AS name, age FROM read_csv('path/to/file.csv')" --json > file.json
```

## Advanced Usage

### Specifying Additional Options

You can specify additional options like headers and delimiters for your CSV file:

```sql
SELECT * FROM read_csv('path/to/file.csv', header=true, delimiter=',');
```

### Error Handling

Make sure to handle errors such as missing files or formatting issues. Anyquery will throw an error if the file does not exist or if there's an issue with the CSV format.

### Performance Considerations

For large CSV files, consider using LIMIT to handle a subset of data at a time:

```sql
SELECT * FROM read_csv('path/to/file.csv') LIMIT 1000;
```

Refer to the [functions documentation](https://anyquery.dev/docs/reference/functions) for more information on the available functions and how to use them.

## Conclusion

You've now learned how to convert a CSV file to a JSON file using Anyquery. This process involves querying the CSV file and exporting the result to JSON format. By using Anyquery, you can perform additional transformations and handle various data sources seamlessly.
