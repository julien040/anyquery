---
title: anyquery run
description: Learn how to use the anyquery run command in Anyquery.
---

Run a SQL query from the community repository

### Synopsis

Run a SQL query from the community repository.
The query can be specified by its ID or by its URL.
If the query is specified by its ID, the query will be downloaded from the repository github.com/julien040/anyquery/tree/queries.
If your query isn't from the repository, you can use the URL to specify the query.

```bash
anyquery run [query_id | http_url | local_path | s3_url] [flags]
```

### Examples

```bash
# Run a query by its ID
anyquery run github_stars_per_day

# Run a query by its URL
anyquery run https://raw.githubusercontent.com/julien040/anyquery/main/queries/github_stars_per_day.sql
```

### Options

```bash
  -c, --config string     Path to the configuration database
      --csv               Output format as CSV
  -d, --database string   Database to connect to (a path or :memory:)
      --format string     Output format (pretty, json, csv, plain)
  -h, --help              help for run
      --in-memory         Use an in-memory database
      --json              Output format as JSON
      --plain             Output format as plain text
      --read-only         Start the server in read-only mode
      --readonly          Start the server in read-only mode
```

### SEE ALSO

* [anyquery](../anyquery)	 - A tool to query any data source
