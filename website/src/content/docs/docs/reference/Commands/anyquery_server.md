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
      --auth-file string    Path to the authentication file
  -c, --config string       Path to the configuration database
  -d, --database string     Database to connect to (a path or :memory:) (default "anyquery.db")
      --dev                 Run the program in developer mode
      --extension strings   Load one or more extensions by specifying their path. Separate multiple extensions with a comma.
  -h, --help                help for server
      --host string         Host to listen on (default "127.0.0.1")
      --in-memory           Use an in-memory database
      --log-file string     Log file (default "/dev/stdout")
      --log-format string   Log format (text, json) (default "text")
      --log-level string    Log level (debug, info, warn, error, fatal) (default "info")
  -p, --port int            Port to listen on (default 8070)
      --readonly            Start the server in read-only mode
```

### SEE ALSO

* [anyquery](../anyquery)	 - A tool to query any data source
