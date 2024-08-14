---
title: Troubleshooting
description: Troubleshooting common issues with the Anyquery CLI
---

Thank you for using anyquery!
Sometimes things don't go as planned. This page will help you troubleshoot common issues with the Anyquery CLI. If you don't find the answer to your question, please [open an issue](https://github.com/julien040/anyquery/issues/new).

Over time, this page will be updated with more common issues and solutions.

## Errors

### Installation 404 with apt-get

If you encounter a 404 error when running `apt-get update` or `apt-get install anyquery`, it's not an issue. Anyquery will still be installed.

### `database locked` error

If you encounter a `database locked` error, it means that you're trying to run multiple write queries at the same time. Anyquery is based on SQLite, which doesn't support concurrent writes. You can run multiple read queries at the same time, but only one write query at a time.

### `another process is using this Badger database. error: resource temporarily unavailable` error

The error `another process is using this Badger database. error: resource temporarily unavailable` means that two plugins are trying to connect to the same cache database at the same time.

It can happen when anyquery was not closed properly, and the zombie processes are still running. It can also happen when you run multiple instances of anyquery at the same time with the same profiles queried.

To fix this issue, you need to kill all the anyquery processes. To do so, open your process manager (e.g., Activity Monitor on macOS or Task Manager on Windows) and kill all the processes named `anyquery.out`, `anyquery.exe`, `anyquery`, or `{plugin-name}*`.

### `no such table: table_name` error

If you encounter a `no such table: table_name` error, it means that the table you're trying to query doesn't exist. You can list all the tables available in the database by running `SHOW TABLES;`.

### `no such column: column_name` error

If you encounter a `no such column: column_name` error, it means that the column you're trying to query doesn't exist. You can list all the columns available in the table by running `DESCRIBE table_name;`.
Note that some column identifiers are reserved. Enclose them in double quotes or backticks to avoid conflicts.

### `no query solution` error

If you encounter a `no query solution` error, it means that the table requires parameters, and they weren't provided. Therefore, SQLite is not able to create a query plan. You need to provide the required parameters to run the query. Refer to the plugin documentation for more information on the required parameters.

## Limitations

Anyquery has several limitations that you should be aware of:

- Anyquery is based on SQLite, which doesn't support concurrent writes. You can run multiple read queries at the same time, but only one write query at a time.
- You cannot run `DESCRIBE`, `WITH`, or `SHOW CREATE TABLE` on file tables. This is due to the way anyquery handles file tables. To observe the schema, you need to create a virtual table as specified in the MySQL server section.
- Concurrent anyquery processes with the same configuration database may lead to unexpected behavior. It's recommended to run only one instance of anyquery at a time for a given configuration database (pass the `-c` flag to specify a different configuration database).
- Dot and slash commands don't work when `PRQL` or `PQL` is enabled. You need to switch to `SQL` mode to run dot and slash commands.
- Anyquery is slow on `ORDER BY` and `GROUP BY` operations. It's due to the need of requesting all the data from the source and then applying the operation. It's recommended to use `LIMIT`, `OFFSET` and `WITH` clauses to reduce the amount of data to process.
- DDL statement on integration tables is not supported. You can only run DML statements on integration tables. The schema must always be defined beforehand (it can be fetched on the fly though).
