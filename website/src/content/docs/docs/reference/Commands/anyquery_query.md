---
title: anyquery query
description: Learn how to use the anyquery query command in AnyQuery.
---

Run a SQL query

### Synopsis

Run a SQL query on the data sources installed on the system.
The query can be specified as an argument or read from stdin.
If no query is provided, the command will launch an interactive input.

```bash
anyquery query [database file] [sql query] [flags]
```

### Examples

```bash
# Run a one-off query
anyquery query -d mydatabase.db -q "SELECT * FROM mytable"

# Open the interactive shell
anyquery query -d mydatabase.db

# Open the interactive shell in memory
anyquery query

# Query from stdin
echo "SELECT * FROM mytable" | anyquery query -d mydatabase.db
```

### Options

```bash
  -c, --config string       Path to the configuration database
      --csv                 Output format as CSV
  -d, --database string     Database to connect to (a path or :memory:)
      --dev                 Run the program in developer mode
      --extension strings   Load one or more extensions by specifying their path. Separate multiple extensions with a comma.
      --format string       Output format (pretty, json, csv, plain)
  -h, --help                help for query
      --in-memory           Use an in-memory database
      --init stringArray    Run SQL commands in a file before the query. You can specify multiple files.
      --json                Output format as JSON
      --language string     Alternative language to use
      --log-file string     Log file
      --log-format string   Log format (text, json) (default "text")
      --log-level string    Log level (trace, debug, info, warn, error, off) (default "info")
      --plain               Output format as plain text
      --pql                 Use the PQL language
      --prql                Use the PRQL language
  -q, --query string        Query to run
      --read-only           Start the server in read-only mode
      --readonly            Start the server in read-only mode
```

### SEE ALSO

* [anyquery](../anyquery)	 - A tool to query any data source
