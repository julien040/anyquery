---
title: anyquery mcp
description: Learn how to use the anyquery mcp command in Anyquery.
---

Start the Model Context Protocol (MCP) server

### Synopsis

Start the Model Context Protocol (MCP) server. It is used to provide context for LLM that supports it. 
Pass the --stdio flag to use standard input/output for communication. By default, it will bind locally to localhost:8070 (modify it with the --host, --port and --domain flags).

```bash
anyquery mcp [flags]
```

### Options

```bash
  -c, --config string       Path to the configuration database
  -d, --database string     Database to connect to (a path or :memory:)
      --domain string       Domain to use for the HTTP tunnel (empty to use the host)
      --extension strings   Load one or more extensions by specifying their path. Separate multiple extensions with a comma.
  -h, --help                help for mcp
      --host string         Host to bind to (default "127.0.0.1")
      --in-memory           Use an in-memory database
      --log-file string     Log file
      --log-format string   Log format (text, json) (default "text")
      --log-level string    Log level (trace, debug, info, warn, error, off) (default "info")
      --port int            Port to bind to (default 8070)
      --read-only           Open the SQLite database in read-only mode
      --readonly            Open the SQLite database in read-only mode
      --stdio               Use standard input/output for communication
      --tunnel              Use an HTTP tunnel, and expose the server to the internet (when used, --host, --domain and --port are ignored)
```

### SEE ALSO

* [anyquery](../anyquery)	 - A tool to query any data source
