---
title: Get started
description: Run SQL queries on remote databases
---

Anyquery is a query engine that can be used to query data from different sources. While it's often used to query data from APIs and files, it can also be used to query data **from databases** (such as PostgreSQL, MySQL, SQLite, etc.) starting from version 0.3.0.

On startup, it can also fetch the list of tables and views from the database and import them automatically.

```sql
SELECT * 
FROM pg.my_table 
JOIN mysql.my_table 
ON pg.my_table.id = mysql.my_table.id
```

## Supported databases

Anyquery supports the following databases:

- PostgreSQL (and perhaps other databases that support the PostgreSQL wire protocol)
- MySQL (and perhaps other databases that support the MySQL wire protocol)
- SQLite

## Connecting to a database

Most of the time, you want to import several tables and views from a database. To do this, run the following command:

```bash
anyquery connection add
```

The CLI will prompt you a few information about the connection:

- Name: A name for the connection. Must be unique. Please mind that the name is used to reference the connection in the queries.
For example, if you name the connection `pg`, you can reference the tables and views from the connection with `pg.my_table`.
- Type: The type of the database. Currently, only `PostgreSQL`, `MySQL`, and `SQLite` are supported.
- Connection string: The connection string to the database. Please refer to the documentation of the database connector for more information. For example, for PostgreSQL, the connection string is `postgresql://user:password@host:port/database`. MySQL uses `mysql://user:password@tcp(host:port)/database`. Make sure to set the database name in the connection string.
- Filter: A CEL script to filter the tables and views to import. For example, to import only the tables that start with `my_`, you can use the following filter: `table.name.startsWith("my_")`. If you don't want to filter the tables, you can leave this field empty. Refer to the [CEL syntax documentation](cel-script) for more information.

Press enter to add the connection. On the next startup, Anyquery will fetch the list of tables and views from the database and import them automatically.

If you don't have a terminal, you can pass the same information in the same order as the command line arguments:

```bash
anyquery connection add <name> <type> <connection_string> [filter]
```

Once done, you can run queries on the tables and views from the database.

```sql
SELECT * FROM pg.my_table;
```

To manage the connections, you can use the following commands:

```bash
# List all connections
anyquery connection list

# Remove a connection
anyquery connection remove <name>

## Documentation for the connection commands
anyquery connection -h
```

## Manually importing a single table or view

If you want to import a single table, you can run the following query:

```sql
-- MySQL
CREATE VIRTUAL TABLE my_table USING mysql_reader('connection_string', 'table_name');

-- PostgreSQL
CREATE VIRTUAL TABLE my_table USING postgresql_reader('connection_string', 'table_name');

-- SQLite
ATTACH DATABASE 'connection_string' AS my_db;
```

Replace `connection_string` with the connection string to the database and `table_name` with the name of the table you want to import.

## Additional information

- [CEL filtering](cel-script): To avoid importing all tables of a database, you can filter them using CEL scripts.
- [anyquery connection](../reference/Commands/anyquery_connection): Learn how to manage connections to other databases.
- [MySQL connector](mysql): Learn how to connect to a MySQL database.
- [PostgreSQL connector](postgresql): Learn how to connect to a PostgreSQL database.
- [SQLite connector](sqlite): Learn how to connect to another SQLite database.
