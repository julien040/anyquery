---
title: anyquery gpt
description: Learn how to use the anyquery gpt command in Anyquery.
---

Open an HTTP server so that ChatGPT can do function calls

### Synopsis

Open an HTTP server so that ChatGPT can do function calls. By default, it will expose a tunnel to the internet.

By setting the --host or --port flags, you can disable the tunnel and bind to a specific host and port. In this case, you will need to configure your LLM to connect to this host and port.
It will also enable the authorization token mechanism. By default, the token is randomly generated and can be found when starting the server. You can also provide a token using the ANYQUERY_AI_SERVER_BEARER_TOKEN environment variable.
This token must be supplied in the Authorization header of the request (prefixed with "Bearer "). You can also disable the authorization mechanism by setting the --no-auth flag.

```bash
anyquery gpt [flags]
```

### Options

```bash
  -c, --config string       Path to the configuration database
  -d, --database string     Database to connect to (a path or :memory:)
      --extension strings   Load one or more extensions by specifying their path. Separate multiple extensions with a comma.
  -h, --help                help for gpt
      --host string         Host to bind to. If not empty, the tunnel will be disabled
      --in-memory           Use an in-memory database
      --log-file string     Log file
      --log-format string   Log format (text, json) (default "text")
      --log-level string    Log level (trace, debug, info, warn, error, off) (default "info")
      --no-auth             Disable the authorization mechanism for locally bound servers
      --port int            Port to bind to. If not empty, the tunnel will be disabled
      --read-only           Open the SQLite database in read-only mode
      --readonly            Open the SQLite database in read-only mode
```

### SEE ALSO

* [anyquery](../anyquery)	 - A tool to query any data source
