---
title: Convert a CSV file to JSON
description: Learn how to convert a CSV file to a JSON file using Anyquery.
---

Anyquery is a SQL query engine that enables you to execute SQL queries on virtually anything, including CSV files. Moreover, as it can export a query result to a JSON file, you can transform a CSV file into a JSON file with a straightforward SQL query.

```bash
anyquery -q "SELECT * FROM read_csv('path/to/file.csv')" --json > file.json
```

Additionally, you can modify each column using functions such as `upper`. For example, the following query converts a CSV file to a JSON file and converts the `name` column to uppercase:

```bash
anyquery -q "SELECT upper(name), age FROM read_csv('path/to/file.csv')" --json > file.json
```

See the [functions documentation](/docs/reference/functions) for more information on the available functions.

:::tip
Because CSV is just a text file, all columns are of text type. If you want to convert a column to a number, you can use the `CAST` function. For example, to convert the `age` column to an integer, you can use `CAST(age AS INT)`.
:::
