---
title: anyquery server
description: Learn how to use the anyquery server command in Anyquery.
---

Lets you connect to anyquery remotely

### Synopsis

Listens for incoming connections and allows you to run queries
using any MySQL client.

```bash
anyquery server [flags]
```

### Examples

```bash
# Start the server by opening anyquery.db by default
anyquery server 

# Start the server on a specific host and port
anyquery server --host 127.0.0.1 --port 3306

# Start the server with a specific database
anyquery server -d mydatabase.db

# Increase the log level and redirect the output to a file
anyquery server --log-level debug --log-file /var/log/anyquery.log
```

### Options

```bash
      --allow-attach           When sandboxed, allow ATTACH/VACUUM INTO to on-disk paths within --allow-dirs
      --allow-db-connections   When sandboxed, allow the database reader modules (duckdb/postgres/mysql/clickhouse/cassandra)
      --allow-dirs strings     When sandboxed, directories that read_* tables (and on-disk ATTACH) may access (repeatable, comma-separated)
      --allow-remote           When sandboxed, allow read_* tables to fetch remote URLs (http/https)
      --auth-file string       Path to the authentication file
  -c, --config string          Path to the configuration database
  -d, --database string        Database to connect to (a path or :memory:) (default "anyquery.db")
      --dev                    Run the program in developer mode (implies --no-sandbox: UNSAFE, exposes local file read, SSRF, and arbitrary file write; do not use on a network-exposed server)
      --extension strings      Load one or more extensions by specifying their path. Separate multiple extensions with a comma.
  -h, --help                   help for server
      --host string            Host to listen on (default "127.0.0.1")
      --in-memory              Use an in-memory database
      --log-file string        Log file (default "/dev/stdout")
      --log-format string      Log format (text, json) (default "text")
      --log-level string       Log level (debug, info, warn, error, fatal) (default "info")
      --no-sandbox             Disable server sandboxing entirely (UNSAFE: exposes local file read, SSRF, and arbitrary file write)
  -p, --port int               Port to listen on (default 8070)
      --readonly               Start the server in read-only mode
```

### SEE ALSO

* [anyquery](../anyquery)	 - A tool to query any data source
