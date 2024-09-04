---
title: "How to join two CSV files?"
description: "Learn how to join two CSV files using Anyquery's SQL capabilities, including reading files, creating virtual tables, filtering data, and exporting results."
---

# How to Join Two CSV Files

Joining two CSV files in Anyquery is straightforward using SQL. In this tutorial, we'll demonstrate how to join two CSV files and perform queries on the joined data. We will also cover filtering, data manipulation, and exporting the results.

## Introduction to Anyquery

Anyquery is a SQL query engine that allows you to run SQL queries on various data sources, including CSV files. It leverages SQLite's virtual table mechanism to extend SQL capabilities. You can install Anyquery by following the installation guide [here](https://anyquery.dev/docs/#installation).

## Prerequisites

Before starting, ensure you have Anyquery installed and two CSV files to join. For example, let's assume we have the following CSV files:

- `employees.csv`:
  ```csv
  employee_id,first_name,last_name,department_id
  1,John,Doe,101
  2,Jane,Doe,102
  3,Jim,Beam,101
  ```

- `departments.csv`:
  ```csv
  department_id,department_name
  101,HR
  102,Finance
  103,Engineering
  ```

## Step 1: Reading the CSV Files

First, let's read the CSV files into Anyquery. You can use the `read_csv` function to read CSV files. Ensure that the files are accessible from the terminal.

```sql
SELECT * FROM read_csv('path/to/employees.csv', header=true);
SELECT * FROM read_csv('path/to/departments.csv', header=true);
```

## Step 2: Creating Virtual Tables

Create virtual tables to query the CSV files as if they were database tables. This step is necessary if you want to join the files in the context of a MySQL server mode.

```sql
CREATE VIRTUAL TABLE employees USING csv_reader('path/to/employees.csv', header=true);
CREATE VIRTUAL TABLE departments USING csv_reader('path/to/departments.csv', header=true);
```

## Step 3: Joining the CSV Files

Now, let's join the CSV files on the `department_id` column. Use the `INNER JOIN` clause to combine the rows from both tables based on matching `department_id` values.

```sql
SELECT e.employee_id, e.first_name, e.last_name, d.department_name 
FROM employees e
INNER JOIN departments d ON e.department_id = d.department_id;
```

## Step 4: Filtering the Joined Data

You can filter the joined data using the `WHERE` clause. For example, to list employees in the HR department:

```sql
SELECT e.employee_id, e.first_name, e.last_name 
FROM employees e
INNER JOIN departments d ON e.department_id = d.department_id
WHERE d.department_name = 'HR';
```

## Step 5: Exporting the Result

Export the joined and filtered data to a new CSV file. Use the `--csv` flag to specify the output format and redirect the output to a file.

```bash
anyquery -q "SELECT e.employee_id, e.first_name, e.last_name, d.department_name FROM employees e INNER JOIN departments d ON e.department_id = d.department_id" --csv > joined_data.csv
```

## Conclusion

You've successfully joined two CSV files using Anyquery and performed various operations on the joined data. You can now explore, filter, and export the data as needed. For more information on querying files and other functionalities, refer to the [Anyquery documentation](https://anyquery.dev/docs/usage/querying-files).
