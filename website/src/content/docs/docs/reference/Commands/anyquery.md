---
title: anyquery
description: Learn how to use the anyquery command in AnyQuery.
---

A tool to query any data source

### Synopsis

Anyquery allows you to query any data source
by writing SQL queries. It can be extended with plugins

```
anyquery [database] [query] [flags]
```

### Options

```
  -c, --config string       Path to the configuration database
      --csv                 Output format as CSV
  -d, --database string     Database to connect to (a path or :memory:)
      --dev                 Run the program in developer mode
      --extension strings   Load one or more extensions by specifying their path. Separate multiple extensions with a comma.
      --format string       Output format (pretty, json, csv, plain)
  -h, --help                help for anyquery
      --in-memory           Use an in-memory database
      --init stringArray    Run SQL commands in a file before the query. You can specify multiple files.
      --json                Output format as JSON
      --language string     Alternative language to use
      --log-file string     Log file
      --log-format string   Log format (text, json) (default "text")
      --log-level string    Log level (trace, debug, info, warn, error, off) (default "info")
      --no-input            Do not launch an interactive input
      --plain               Output format as plain text
      --pql                 Use the PQL language
      --prql                Use the PRQL language
  -q, --query string        Query to run
      --read-only           Start the server in read-only mode
      --readonly            Start the server in read-only mode
  -v, --version             Print the version of the program
```

### SEE ALSO

* [anyquery alias](../anyquery_alias)	 - Manage the aliases
* [anyquery completion](../anyquery_completion)	 - Generate the autocompletion script for the specified shell
* [anyquery install](../anyquery_install)	 - Search and install a plugin
* [anyquery plugins](../anyquery_plugins)	 - Print the plugins installed on the system
* [anyquery profiles](../anyquery_profiles)	 - Print the profiles installed on the system
* [anyquery query](../anyquery_query)	 - Run a SQL query
* [anyquery registry](../anyquery_registry)	 - List the registries where plugins can be downloaded
* [anyquery server](../anyquery_server)	 - Lets you connect to anyquery remotely
* [anyquery tool](../anyquery_tool)	 - Tools to help you with using anyquery
