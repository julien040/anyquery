---
title: PostgreSQL
description: Learn how to connect a PostgreSQL database to Anyquery.
---

![MySQL](/icons/postgresql.svg)

Anyquery is able to run queries from PostgreSQL databases. This is useful when you want to import/export data from/to a PostgreSQL database. Or when you want to join an API with a PostgreSQL database.

## Connection

To connect a PostgreSQL database to Anyquery, you need to provide the connection string. The connection string is a URL that contains the necessary information to connect to the database. The connection string has the following format:

```txt
postgresql://user:password@host:port/database
// or
user=your_user password=your_password host=your_host port=your_port dbname=your_db sslmode=disable
```

For example: `postgresql://user:password@localhost:5432/mydb`. Refer to the [DSN documentation](https://www.postgresql.org/docs/current/libpq-connect.html#LIBPQ-CONNSTRING) for more information.

To add a PostgreSQL connection, use the following command:

```bash
anyquery connection add
```

The CLI will ask you for a connection name, the type of the connection (PostgreSQL), and the connection string. You can also provide a filter to import only the tables you need.

- The connection name is used to reference the connection in the queries. For example, if you name the connection `pg`, you can reference the tables and views from the connection with `pg.my_table`.
- The connection string is the URL to the PostgreSQL database. See above for the format.
- The filter is a CEL script that filters the tables and views to import. For example, to import only the tables that start with `my_`, you can use the following filter: `table.name.startsWith("my_")`. If you don't want to filter the tables, you can leave this field empty. Refer to the [CEL syntax documentation](cel-script) for more information.

Press enter to add the connection. On the next startup, Anyquery will fetch the list of tables and views from the database and import them automatically.

Congratulations 🎉! You can now run queries on the tables and views from the PostgreSQL database. The table name will follow this format `connection_name.[schema_name]_table_name` (e.g. `pg.information_schema_tables`)

```sql
-- Join a table from PostgreSQL with a table from an API
SELECT * 
FROM pg.my_table m
JOIN github_stargazers_from_repository('julien040/anyquery') g
ON m.id = g.login

-- Insert data into a PostgreSQL table
BEGIN;
INSERT INTO pg.my_table (id, name) VALUES (1, 'John Doe');

-- Update data in a PostgreSQL table
UPDATE pg.my_table SET name = 'Jane Doe' WHERE id = 1;

-- Delete data from a PostgreSQL table
DELETE FROM pg.my_table WHERE id = 1;
COMMIT;

-- List all tables of the connection
SELECT * FROM pg.information_schema_tables;
```

## Additional information

### Types

Anyquery only supports `TEXT`, `INT`, `DOUBLE`, and `BLOB` types. Due to this limitation, Anyquery automatically converts the PostgreSQL types to the supported types.

| PostgreSQL type               | Anyquery type | Additional information                                                            |
| ----------------------------- | ------------- | --------------------------------------------------------------------------------- |
| `smallint`                    | `INT`         |                                                                                   |
| `integer`                     | `INT`         |                                                                                   |
| `bigint`                      | `INT`         |                                                                                   |
| `bit`                         | `INT`         | Insert/update value as its binary representation e.g., `'101'`                    |
| `real`                        | `DOUBLE`      |                                                                                   |
| `double precision`            | `DOUBLE`      |                                                                                   |
| `numeric`                     | `DOUBLE`      |                                                                                   |
| `smallserial`                 | `INT`         |                                                                                   |
| `serial`                      | `INT`         |                                                                                   |
| `bigserial`                   | `INT`         |                                                                                   |
| `text`                        | `TEXT`        |                                                                                   |
| `character varying`           | `TEXT`        |                                                                                   |
| `character`                   | `TEXT`        |                                                                                   |
| `bytea`                       | `BLOB`        |                                                                                   |
| `json`                        | `TEXT`        |                                                                                   |
| `jsonb`                       | `TEXT`        |                                                                                   |
| `date`                        | `TEXT`        |                                                                                   |
| `time`                        | `TEXT`        |                                                                                   |
| `timestamp`                   | `TEXT`        | converted to RFC3339 format (similar to `ISO8601`)                                |
| `timestamp with time zone`    | `TEXT`        |                                                                                   |
| `timestamp without time zone` | `TEXT`        |                                                                                   |
| `interval`                    | `TEXT`        |                                                                                   |
| `boolean`                     | `INT`         | `0` for `false`, `1` for `true`                                                   |
| `enum`                        | `TEXT`        | returns the value as a string                                                     |
| `uuid`                        | `TEXT`        | returns the string representation of the UUID                                     |
| `inet`                        | `TEXT`        |                                                                                   |
| `cidr`                        | `TEXT`        |                                                                                   |
| `macaddr`                     | `TEXT`        |                                                                                   |
| `macaddr8`                    | `TEXT`        |                                                                                   |
| `array(T)`                    | `TEXT`        | Inserted as `'{1,2,3}'`, returned as `[1,2,3]`                                    |
| `point`                       | `TEXT`        | As with all geometry types, it's converted to its text form <br> ⚠️ Not filterable |
| `line`                        | `TEXT`        | See above                                                                         |
| `lseg`                        | `TEXT`        | See above                                                                         |
| `box`                         | `TEXT`        | See above                                                                         |
| `path`                        | `TEXT`        | See above                                                                         |
| `polygon`                     | `TEXT`        | See above                                                                         |
| `circle`                      | `TEXT`        | See above                                                                         |

Other types might be supported, but they are not tested. If you encounter an issue with a specific type, feel free to open an issue on the GitHub repository.

To handle `JSON` types, Anyquery converts the JSON to a string. You can then use the [many JSON functions](https://www.sqlite.org/json1.html) available in SQLite. For example:

```sql
SELECT column ->> '$.key' FROM pg.my_table;
SELECT json_array_length(column) FROM pg.my_table;
```

Geometry types are also converted to their text representation. Refer to the [PostgreSQL documentation](https://www.postgresql.org/docs/current/datatype-geometric.html#DATATYPE-GEO-TABLE) for more information.

### Functions

When querying a PostgreSQL database, functions aren't passed to the PostgreSQL server. Instead, they are executed in the SQLite database. It might often result in a performance hit. To avoid this, you can create a view in PostgreSQL that computes the function and then query the view in Anyquery.

### Transactions support

We'll not claim that Anyquery is [ACID](https://en.wikipedia.org/wiki/ACID?useskin=vector) compliant. However, Anyquery supports transactions. You can start a transaction with `BEGIN;`, commit it with `COMMIT;`, or rollback it with `ROLLBACK;`.

```sql
BEGIN;
INSERT INTO pg.my_table (id, name) VALUES (1, 'John Doe');
UPDATE pg.my_table SET name = 'Jane Doe' WHERE id = 1;
ROLLBACK;

-- No data is inserted or updated
```

### Connection pooling

Anyquery uses a connection pool to connect to the PostgreSQL database. We try to use as little connections as possible to avoid exhausting the PostgreSQL server. However, note that any started transaction consumes a connection until it's committed or rolled back.
A connection might be reused between several tables or views. It's not possible to control which connection is used for a specific table or view.

### Composite primary keys

Due to limitations of SQLite (the underlying query engine), we're unable to provide INSERT/UPDATE/DELETE support for tables with composite primary keys. However, you can still query the table using SELECT queries. If this an issue for you, feel free to open an issue on the GitHub repository.
