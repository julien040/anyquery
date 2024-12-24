---
title: "How to use PRQL to query a SQLite database?"
description: "Learn how to use PRQL with Anyquery for querying SQLite databases, offering readable syntax and intuitive data manipulation. Includes setup, examples, and exporting results."
---

# How to Use PRQL to Query a SQLite Database

Anyquery is a SQL query engine that allows you to run SQL queries on pretty much anything, including SQLite databases. Additionally, Anyquery supports alternative query languages such as PRQL (Pipelined Relational Query Language). PRQL aims to offer a more human-readable and expressive syntax compared to traditional SQL. In this guide, we'll learn how to use PRQL to query a SQLite database.

## Prerequisites

Before starting, ensure you have the following:

- A working [installation of Anyquery](https://anyquery.dev/docs/#installation).

## Introduction to PRQL

PRQL is a language designed to make SQL queries more readable and maintainable. The main difference is that PRQL uses a series of functions that transform data in a pipeline, offering a more intuitive approach to data manipulation.

### Example of PRQL

Here is a basic example:

```plain
from employees
filter salary > 50000
select { employee_id, name, salary }
sort salary
take 10
```

The above PRQL query translates to the following SQL:

```sql
SELECT employee_id, name, salary
FROM employees
WHERE salary > 50000
ORDER BY salary
LIMIT 10;
```

## Enabling PRQL in Anyquery

First, install the `prqlc` CLI tool by following the instructions on the [PRQL website](https://prql-lang.org/book/project/integrations/prqlc-cli.html#installation).

Then, let's open Anyquery in PRQL mode. You can do this by passing the `--prql` flag when starting Anyquery:

```bash
anyquery --prql
```

Alternatively, you can switch to PRQL mode from within the Anyquery shell:

```sql
.language prql
```

## Running PRQL Queries

Now that we're in PRQL mode, let's look at some basic operations.

### Select and Filter

Suppose you have a SQLite database named `company.db` with a table named `employees`. You can run the following PRQL query to select and filter data:

```plain
from employees
filter salary > 50000
select { employee_id, name, salary }
```

### Sorting and Limiting

You can also sort and limit the data:

```plain
from employees
filter salary > 50000
select { employee_id, name, salary }
sort salary
take 10
```

### Joining Tables

If you have another table named `departments`, you can join it with the `employees` table:

```plain
from employees
join departments [ department_id ]
select { employees.employee_id, employees.name, departments.department_name }
```

## Full Example

Let’s say we want to query the top 5 highest-paid employees along with their department names. Here is how you can do it using PRQL:

```plain
from employees
join departments [ department_id ]
filter employees.salary > 50000
select { employees.employee_id, employees.name, employees.salary, departments.department_name }
sort employees.salary
take 5
```

## Exporting Results

You can export the result of a PRQL query to various formats such as JSON, CSV, etc. For example, to export to a CSV file:

```bash
anyquery --prql -q "from employees filter salary > 50000 select { employee_id, name, salary }" --csv > employees.csv
```

## Conclusion

Using PRQL with Anyquery provides a more intuitive and human-readable way to interact with your SQLite databases. You can perform complex queries with ease and export the results in various formats. For more information, refer to the [Anyquery documentation](https://anyquery.dev/docs/).

Feel free to explore more about PRQL and its capabilities on the [PRQL official website](https://prql-lang.org).
