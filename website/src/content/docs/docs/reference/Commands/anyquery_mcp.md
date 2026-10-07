---
title: anyquery mcp
description: Learn how to use the anyquery mcp command in Anyquery.
---

Start the Model Context Protocol (MCP) server

### Synopsis

Start the Model Context Protocol (MCP) server. It is used to provide context for LLM that supports it. 
Pass the --stdio flag to use standard input/output for communication. 

By default, it will bind locally to localhost:8070 (modify it with the --host, --port and --domain flags). The server exposes two HTTP endpoints on the same port:
- /mcp for the Streamable HTTP transport (recommended)
- /sse for the legacy server-sent events (SSE) transport

Supported MCP protocol versions: 2026-07-28, 2025-11-25, 2025-06-18, 2025-03-26 and 2024-11-05. The version is negotiated with each client.

Authentication is enabled by default for HTTP connections. The token can be found when starting the server, or you can provide one using the ANYQUERY_AI_SERVER_BEARER_TOKEN environment variable.
You can disable the authorization mechanism by setting the --no-auth flag.

```bash
anyquery mcp [flags]
```

### Options

```bash
      --allow-attach           When sandboxed, allow ATTACH/VACUUM INTO to on-disk paths within --allow-dirs
      --allow-db-connections   When sandboxed, allow the database reader modules (duckdb/postgres/mysql/clickhouse/cassandra)
      --allow-dirs strings     When sandboxed, directories that read_* tables (and on-disk ATTACH) may access (repeatable, comma-separated)
      --allow-remote           When sandboxed, allow read_* tables to fetch remote URLs (http/https)
  -c, --config string          Path to the configuration database
  -d, --database string        Database to connect to (a path or :memory:)
      --domain string          Domain to use for the HTTP tunnel (empty to use the host)
      --extension strings      Load one or more extensions by specifying their path. Separate multiple extensions with a comma.
  -h, --help                   help for mcp
      --host string            Host to bind to (default "127.0.0.1")
      --in-memory              Use an in-memory database
      --log-file string        Log file
      --log-format string      Log format (text, json) (default "text")
      --log-level string       Log level (trace, debug, info, warn, error, off) (default "info")
      --no-auth                Disable the authorization mechanism for locally bound HTTP servers
      --port int               Port to bind to (default 8070)
      --read-only              Open the SQLite database in read-only mode
      --readonly               Open the SQLite database in read-only mode
      --sandbox                Apply server-style sandboxing restrictions (off by default in CLI mode)
      --stdio                  Use standard input/output for communication
      --tunnel                 Use an HTTP tunnel, and expose the server to the internet (when used, --host, --domain and --port are ignored)
```

### SEE ALSO

* [anyquery](../anyquery)	 - A tool to query any data source
