---
title: Convert a JSON file to CSV
description: Learn how to convert a JSON file to a CSV file using Anyquery.
---

Anyquery is a SQL query engine that enables you to execute SQL queries on virtually anything, including JSON files. Moreover, as it can export a query result to a CSV file, you can transform a JSON file into a CSV file with a straightforward SQL query.

```bash
anyquery -q "SELECT * FROM read_json('path/to/file.json')" --csv > file.csv
```

Additionally, you can modify each column using functions such as `upper`. For example, the following query converts a JSON file to a CSV file and converts the `name` column to uppercase:

```bash
anyquery -q "SELECT upper(name), age FROM read_json('path/to/file.json')" --csv > file.csv
```

See the [functions documentation](/docs/reference/functions) for more information on the available functions.
