---
title: SQLite
description: Learn how to connect a SQLite database to Anyquery.
---

![SQLite](/icons/sqlite.svg)

Anyquery is based on SQLite. Therefore, you can easily attach a SQLite database to Anyquery.

## Connection string

The connection string for SQLite has the following format:

```txt
file:/path/to/database.db?option1=value1&option2=value2
```

### Connection options

- `file`: The path to the SQLite database file.
- `mode`: The mode to open the database file. The default is `rw` (read-write). You can also use `ro` (read-only) or `memory` (in-memory).
- `immutable`: If set to `true`, the database file is opened in immutable mode. This means that the database file is read-only and cannot be modified.

## Attach a SQLite database

You have two options to attach a SQLite database to Anyquery:

- Attach it on startup by registering a connection in the configuration database.
- Run the `ATTACH DATABASE` command in the shell (or MySQL client).

### Attach on startup

To attach a SQLite database on startup, you need to register a connection in the configuration database. You can do this by running the following command:

```bash
anyquery connection add
```

Then, provide the connection string, SQLite as the database type and a name for the connection. The name is used to reference the connection in the queries. For example, if you provide the name `mydb`, you can reference the connection in a query like this:

```sql
SELECT * FROM mydb.mytable;
```

The [CEL filter](../cel-schema) will be ignored when querying the attached SQLite database.

### Attach in the shell

To attach a SQLite database in the shell, you can run the `ATTACH DATABASE` command. For example, to attach a database located at `/path/to/database.db`, you can run the following command:

```sql
ATTACH DATABASE file:/path/to/database.db?mode=ro AS mydb;
```

Then, you can reference the attached database in a query like this:

```sql
SELECT * FROM mydb.mytable;
```
