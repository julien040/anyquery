---
title: Export a SQLite database to HTML
description: Learn how to export a SQLite database to an HTML file using Anyquery.
---

Anyquery is a SQL query engine that enables you to execute SQL queries on virtually anything, including SQLite databases. Moreover, as it can export a query result to an HTML table, you can transform a SQLite database into an HTML file with a straightforward SQL query.

```bash
anyquery -q "SELECT * FROM sqlite_master" --format html > tables.html
```

Additionally, you can modify each column using functions such as `upper`. For example, the following query exports a SQLite database to an HTML file and converts the `name` column to uppercase:

```bash
anyquery -q "SELECT upper(name), sql FROM sqlite_master" --format html > tables.html
```

See the [functions documentation](/docs/reference/functions) for more information on the available functions.
