---
title: Clickhouse
description: Learn how to connect a Clickhouse database to Anyquery.
---

<img src="/icons/clickhouse.svg" alt="Clickhouse" width="100" height="100">

Anyquery is able to run queries from ClickHouse databases. This is useful when you want to import/export data from/to a ClickHouse database. Or when you want to join an API with a ClickHouse database.

## Connection

To connect a ClickHouse database to Anyquery, you need to provide the connection string. The connection string is a URL that contains the necessary information to connect to the database. The connection string has the following format:

```txt
clickhouse://username:password@host:port/database?param1=value1&param2=value2
```

For example: `clickhouse:mypassword@127.0.0.1:9000/default`. Refer to the [DSN documentation](https://github.com/ClickHouse/clickhouse-go#dsn) for more information.

To add a ClickHouse connection, use the following command:

```bash
anyquery connection add
```

The CLI will ask you for a connection name, the type of the connection (ClickHouse), and the connection string. You can also provide a filter to import only the tables you need.

- The connection name is used to reference the connection in the queries. For example, if you name the connection `ClickHouse`, you can reference the tables and views from the connection with `clickhouse.my_table`.
- The connection string is the URL to the ClickHouse database. See above for the format.
- The filter is a CEL script that filters the tables and views to import. For example, to import only the tables that start with `my_`, you can use the following filter: `table.name.startsWith("my_")`. If you don't want to filter the tables, you can leave this field empty. Refer to the [CEL syntax documentation](cel-script) for more information.

Press enter to add the connection. On the next startup, Anyquery will fetch the list of tables and views from the database and import them automatically.

Congratulations 🎉! You can now run queries on the tables and views from the ClickHouse database. The table name will follow this format `connection_name.[schema_name]_table_name` (e.g. `clickhouse.information_schema_tables`)

```sql
-- Join a table from ClickHouse with a table from an API
SELECT * 
FROM clickhouse.my_table m
JOIN github_stargazers_from_repository('julien040/anyquery') g
ON m.id = g.login

-- Insert data into a ClickHouse table
INSERT INTO clickhouse.my_table (id, name) VALUES (1, 'John Doe');

-- List all tables of the connection
SELECT * FROM clickhouse.information_schema_tables;
```

## Additional information

### Functions

When querying a ClickHouse database, functions aren't passed to the ClickHouse server. Instead, they are executed in the SQLite database. It might often result in a performance hit. To avoid this, you can create a view in ClickHouse that computes the function and then query the view in Anyquery.

### Performance

ClickHouse is designed for high performance. While this integration tries to transfer the least amount of data possible, any query that requires a join or a WHERE invoquing a function will result in all the rows transferred to Anyquery, which can be slow. To mitigate this, you can create views in ClickHouse that pre-compute the necessary data and then query those views in Anyquery.

### Update/Delete support

ClickHouse does not require a primary key for tables. This conflicts with the data model used by Anyquery, which requires a primary key to reference rows in a table, and later update or delete them. As a result, Anyquery does not support updating or deleting rows in ClickHouse tables. You can still insert new rows.

Pull requests are welcome to add support for this feature, as I don't see a workaround for this limitation.

### Types

ClickHouse supports a wide range of types. Anyquery will try to map the types to the closest equivalent in SQLite. Integers are converted to `INT`, floating point numbers to `DOUBLE`, and strings to `TEXT`. Maps, arrays, tuples and JSON are converted to `JSON`. Refer to the [JSON documentation](/docs/usage/working-with-arrays-objects-json) for more information on how to work with JSON in Anyquery. Nested types are converted to a column for each field in the nested type. Enums are converted to `TEXT`. Dates and datetimes are converted to `DATETIME` using the `RFC3339` format.
