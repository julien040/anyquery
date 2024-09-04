---
title: Import a Parquet file into SQLite
description: Learn how to import a Parquet file into SQLite using Anyquery.
---

Anyquery is a SQL query engine that enables you to execute SQL queries on virtually anything, including Parquet files. Under the hood, Anyquery uses [SQLite](https://www.sqlite.org/index.html) as the storage engine, which allows you to import Parquet files into SQLite with a straightforward SQL query.

```bash
anyquery -q "CREATE TABLE parquet AS SELECT * FROM read_parquet('path/to/file.parquet')"
```

Additionally, you can modify each column using functions such as `upper`. For example, the following query imports a Parquet file into SQLite and converts the `name` column to uppercase:

```bash
anyquery -q "CREATE TABLE parquet AS SELECT upper(name), age FROM read_parquet('path/to/file.parquet')"
```

See the [functions documentation](/docs/reference/functions) for more information on the available functions.
