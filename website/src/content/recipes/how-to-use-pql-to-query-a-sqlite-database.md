---
title: "How to use PQL to query a SQLite database?"
description: "Learn to use PQL with Anyquery to query a SQLite database. Set up Anyquery, enable PQL, and run various queries like filtering, sorting, and aggregating data efficiently."
---

# How to Use PQL to Query a SQLite Database

Anyquery is a SQL query engine that allows you to run SQL queries on virtually anything. One of its strengths is the ability to use alternative query languages, such as PQL (Portable Query Language), to query data. In this tutorial, you will learn how to use PQL to query a SQLite database.

## Introduction to Anyquery

Anyquery uses SQLite as its core query engine and can connect to various data sources through plugins. With Anyquery, you can query data from databases, APIs, and even files. You can install Anyquery by following the instructions on the [installation page](https://anyquery.dev/docs/#installation).

Here's an example of querying a GitHub repository using SQL:

```sql
SELECT * FROM github_repositories_from_user('cloudflare') ORDER BY stargazers_count DESC;
```

## Setting Up Anyquery with SQLite

Before we dive into PQL, let's set up Anyquery to work with a SQLite database:

1. **Install Anyquery**: Follow the [installation guide](https://anyquery.dev/docs/#installation).
2. **Create a SQLite Database**: You can create a SQLite database using SQLite command line or any SQLite GUI tool. For this example, let's assume you have a database file named `example.db`.

## Enabling PQL in Anyquery

PQL (Portable Query Language) is a language similar to Microsoft Kusto Query Language (KQL). To enable PQL in Anyquery, run the following command:

```bash
anyquery --pql
```

Alternatively, you can switch to PQL after entering the shell mode by running:

```sql
.language pql
```

To switch back to SQL, run:

```sql
.language
```

Refer to the [alternative languages documentation](https://anyquery.dev/docs/usage/alternative-languages) for more details.

## Running PQL Queries on SQLite

Once PQL is enabled, you can start querying your SQLite database using PQL syntax. Here are some examples:

### Example 1: Selecting All Rows from a Table

```plain
example_table
| project *
```

### Example 2: Filtering Rows

```plain
example_table
| where age > 25
| project name, age
```

### Example 3: Sorting Rows

```plain
example_table
| sort by age desc
| project name, age
```

### Example 4: Limiting Rows

```plain
example_table
| take 10
| project name, age
```

### Example 5: Aggregating Data

```plain
example_table
| summarize avg_age=avg(age) by department
```

### Example 6: Joining Tables

Note: PQL does not directly support joins like SQL. To achieve this, you can use subqueries or CTEs.

```plain
with subquery1 as (
    select * from employees
),
subquery2 as (
    select * from departments
)
subquery1
| join subquery2 on subquery1.department_id == subquery2.id
| project subquery1.name, subquery2.department_name
```

## Exporting Results

You can export query results to various formats such as JSON, CSV, etc. For example, to export the results to a JSON file:

```bash
anyquery -q "example_table | project * | take 10" --json > output.json
```

Refer to the [exporting results documentation](https://anyquery.dev/docs/usage/exporting-results) for more details.

## Conclusion

You have learned how to use PQL to query a SQLite database with Anyquery. PQL offers a readable and intuitive syntax for querying data, making it a powerful alternative to SQL. Explore more features and plugins in the [official documentation](https://anyquery.dev/docs/usage/) to unlock the full potential of Anyquery.
