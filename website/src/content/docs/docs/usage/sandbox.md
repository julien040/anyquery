---
title: Sandboxing
description: Restrict what SQL clients can read, fetch and write when anyquery is exposed as a server.
---

When anyquery is exposed to clients, as a [MySQL server](/docs/usage/mysql-server) or as an [LLM endpoint](/docs/usage/connecting-llms) (`gpt`/`mcp`), those clients can run arbitrary SQL. Anyquery's built-in features make arbitrary SQL powerful: the `read_*` table functions read files, several of them fetch remote URLs, `ATTACH DATABASE` writes files, and a few scalar functions read files or delete cache directories. On a server, that turns into local file read, server-side request forgery (SSRF), and arbitrary file write for anyone who can reach the port.

The sandbox closes those doors. It is a policy attached to the database namespace that confines file access to an explicit set of directories, blocks remote fetches, and denies the dangerous SQL statements and functions.

:::caution[Security fix]
The sandbox was introduced to address [CVE-2026-50006](https://www.cve.org/CVERecord?id=CVE-2026-50006) and [CVE-2026-47253](https://www.cve.org/CVERecord?id=CVE-2026-47253) (local file read, SSRF and arbitrary file write through unrestricted virtual-table modules and SQL functions in server mode). It is **enabled by default** on every network-facing surface. Keep it on in production, and add [authentication](/docs/usage/mysql-server#adding-authentication) on top of it.
:::

## When is the sandbox active?

The default depends on the command, because the exposure differs:

| Command | Sandbox by default | How to change it |
| --- | --- | --- |
| `anyquery server` | **On** | `--no-sandbox` to disable, or `--dev` (see below) |
| `anyquery gpt` | **On** (exposes an internet tunnel by default) | `--no-sandbox` to disable |
| `anyquery mcp` | **Auto**: on when network-exposed (`--tunnel`, or a non-loopback `--host`); off for plain `localhost`/`--stdio` | `--sandbox` to force on, `--sandbox=false` to force off |
| `anyquery query` / interactive shell | **Off** (local use is trusted) | `--sandbox` to opt in |

CLI mode is meant for local data analysis and is not an attack surface, so it is unrestricted by default. You can still opt in with `--sandbox` to mirror the server's behaviour.

## What the sandbox restricts

When active, the sandbox enforces the following. The default is **deny everything**, then you relax it with the flags below.

- **File reads**: the `read_*` table functions (`read_csv`, `read_json`, `read_parquet`, `read_yaml`, `read_toml`, `read_jsonl`, `read_html`, `read_log`) may only read files inside the directories you list with `--allow-dirs`. The confinement is enforced by the operating system, and a symlink inside an allowed directory cannot be used to escape it.
- **Remote fetches**: fetching `http` and `https` URLs is disabled unless you pass `--allow-remote`. S3 and GCS URLs are not supported; query a presigned HTTPS URL instead (see [Querying files](/docs/usage/querying-files#remote-files)).
- **stdin**: denied under the sandbox, in every command (`read_csv('stdin')`, `-`, `/dev/stdin`), regardless of `--allow-dirs` or `--allow-remote`.
- **Database readers**: the `duckdb_reader`, `postgres_reader`, `mysql_reader`, `clickhouse_reader` and `cassandra_reader` modules are not registered at all (they take arbitrary connection strings, and DuckDB can itself read local files and load extensions). `CREATE VIRTUAL TABLE … USING duckdb_reader(...)` fails with `no such module` unless you pass `--allow-db-connections`. With that flag, `duckdb_reader` still only opens local database files inside `--allow-dirs` (no remote URLs), and DuckDB runs with its own file and network access turned off, so a query or a view stored in the database cannot read other files.
- **`ATTACH DATABASE` / `VACUUM … INTO`**: both are arbitrary-file-write primitives. In-memory databases (`:memory:`, `mode=memory`) are always allowed; writing to disk is denied unless you pass `--allow-attach`, and even then it is confined to `--allow-dirs`.
- **Blocked SQL functions**: see [below](#blocked-sql-functions).
- **Restricted PRAGMAs**: see [below](#restricted-pragmas).
- **Dot commands** (interactive shell only): `.shell`, `.system`, `.output`, `.log` and `.cd` are refused, because they run programs, write to any path, or change the working directory. The other dot commands (`.tables`, `.mode`, …) keep working.

## Plugins are not sandboxed

:::danger
The sandbox only restricts anyquery's built-in tables and functions. **Installed plugins are loaded under the sandbox too, and the sandbox does not restrict them.** Every client of a sandboxed server can query them with their full access.
:::

Plugins run as separate processes (or as SQLite extensions), so the restrictions above never apply to them. A sandboxed instance still exposes everything the installed plugins can do, for example:

- the `file` plugin lists any directory on the machine, regardless of `--allow-dirs`;
- the `git` plugin reads the history and diffs of any repository on disk;
- SaaS plugins (`github`, `notion`, `google_sheets`, …) query those services with the credentials stored in their profiles;
- extensions such as `sqlite-http` make HTTP requests, regardless of `--allow-remote`.

When the sandbox is on and plugins are installed, anyquery prints a warning listing them at startup. Before exposing a server, uninstall every plugin you do not want its clients to use (`anyquery plugins uninstall <plugin>`), or give the server its own configuration database that only holds the plugins you intend to expose:

```bash title="A server that only exposes the github plugin"
anyquery install -c /srv/anyquery/config.db github
anyquery server -c /srv/anyquery/config.db
```

## Relaxing the restrictions

The default configuration is intentionally strict: no readable directories, no remote access, no database connections. Open up only what you need.

```bash title="Allow read_* tables to read two directories"
anyquery server --allow-dirs /var/data,/srv/exports
```

```bash title="Allow remote fetches (http/https)"
anyquery server --allow-dirs /var/data --allow-remote
```

```bash title="Allow ATTACH/VACUUM INTO to disk (still confined to --allow-dirs)"
anyquery server --allow-dirs /var/data --allow-attach
```

```bash title="Re-enable the database reader modules"
anyquery server --allow-db-connections
```

| Flag | Effect |
| --- | --- |
| `--allow-dirs <dir,dir>` | Directories the `read_*` tables (and on-disk `ATTACH`) may access. Repeatable / comma-separated. Empty by default. |
| `--allow-remote` | Allow `read_*` tables to fetch remote URLs. |
| `--allow-attach` | Allow `ATTACH DATABASE` / `VACUUM … INTO` to on-disk paths within `--allow-dirs`. |
| `--allow-db-connections` | Register the `duckdb_reader`/`postgres_reader`/… modules. |

:::note
`--allow-remote` is all-or-nothing: there is no IP or SSRF filtering underneath it (no loopback/RFC1918 blocking), only a scheme check. When enabled, the server can again reach internal addresses and cloud metadata endpoints (e.g. `169.254.169.254`). Only enable it on trusted deployments.
:::

With `--allow-remote`, share links (GitHub, GitLab, Codeberg, Hugging Face, Google Sheets, Dropbox; see [Querying files](/docs/usage/querying-files#remote-files)) are rewritten to their raw-file form before fetching. Operators who want to know the egress surface up front should therefore expect outbound requests to any host a query names directly, plus:

```text
raw.githubusercontent.com, gist.githubusercontent.com, huggingface.co,
docs.google.com, www.dropbox.com, gitlab.com, www.gitlab.com, codeberg.org
```

## Blocked SQL functions

A handful of scalar functions read files or delete directories on disk. When the sandbox is active they are **denied outright**, and they cannot be relaxed with `--allow-dirs`:

| Function | Why it is blocked |
| --- | --- |
| `load_file`, `load_file_bytes` | Read an arbitrary file into a value (a local-file-read bypass of the `read_*` confinement). |
| `clear_plugin_cache`, `clear_file_cache` | Delete cache directories on disk (cache management is an operator action, not a client one). |
| `load_extension` | Loading a SQLite extension is remote code execution. The SQL function is disabled by the driver, and the sandbox denies it explicitly as defence in depth. |

```sql title="Blocked under the sandbox"
SELECT load_file('/etc/passwd');   -- error: not authorized
```

To read a file inside an allowed directory, use a `read_*` table function instead of `load_file`: those are permitted within `--allow-dirs`:

```sql title="Allowed when /var/data is in --allow-dirs"
SELECT * FROM read_csv('/var/data/report.csv');
```

## Restricted PRAGMAs

`PRAGMA` is gated to a **read-only allowlist**. Only schema-introspection pragmas that the engine, the MySQL protocol handler and the `information_schema`/`SHOW` emulation rely on are permitted:

```text
table_info, table_xinfo, table_list, index_info, index_xinfo, index_list,
foreign_key_list, database_list, collation_list, function_list, module_list,
pragma_list, compile_options
```

Every other `PRAGMA` is denied. This blocks schema-corruption vectors (`PRAGMA writable_schema=ON` followed by `UPDATE sqlite_master`) and memory-inflation pragmas (`PRAGMA cache_size`, `PRAGMA mmap_size`).

```sql title="Blocked under the sandbox"
PRAGMA writable_schema=ON;   -- error: not authorized
```

## Disabling the sandbox

The function deny-list and the PRAGMA allowlist are part of the sandbox and cannot be relaxed individually. If you genuinely need `load_file`, an arbitrary `PRAGMA`, or the database readers without restriction, you must turn the sandbox off entirely. Only do this on a trusted, non-exposed deployment:

```bash title="Disable the sandbox (UNSAFE on an exposed port)"
anyquery server --no-sandbox
```

```bash title="MCP exposed via a tunnel, but sandbox explicitly forced off"
anyquery mcp --tunnel --sandbox=false
```

### `--dev` always disables the sandbox

`--dev` registers the developer UDFs (`load_dev_plugin`, `reload_dev_plugin`, `unload_dev_plugin`, see [Creating a plugin](/docs/developers/plugins/create-plugin)) that read a manifest file from an arbitrary path, run its `build_command`, and write to its `log_file`, none of which go through the sandbox's file-access policy. **`--dev` implies `--no-sandbox`**, that means on `anyquery server`, passing `--dev` disables the sandbox entirely, regardless of any other `--allow-*` flags

```bash title="server --dev: sandbox is OFF, logged loudly"
anyquery server --dev
```

:::caution
Never bind a `--dev` server to a non-loopback `--host`. The MySQL server has no authentication by default (see [Adding authentication](/docs/usage/mysql-server#adding-authentication)).
:::

`anyquery query` / the interactive shell are unsandboxed by default already, so `--dev` doesn't change their sandbox posture. If you do pass `--dev --sandbox` together there, the sandbox stays on but the dev UDFs are withheld. Pass `--dev` without `--sandbox` to use them.

## Enabling the sandbox in CLI mode

CLI mode is unrestricted by default. Pass `--sandbox` to apply the same policy, which is useful when running untrusted SQL locally, or to reproduce the server's behaviour:

```bash title="Run a query under the sandbox"
anyquery --sandbox --allow-dirs /var/data -q "SELECT * FROM read_csv('/var/data/report.csv')"
```

Without `--allow-dirs`, a sandboxed query cannot read any file:

```bash
anyquery  --sandbox -q "SELECT * FROM read_csv('/etc/passwd')"
# error: sandbox: access to "/etc/passwd" is not allowed; permitted directories: []
```
