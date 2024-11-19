---
title: MySQL
description: Learn how to connect a MySQL database to Anyquery.
---

![MySQL](/icons/mysql.svg)

Anyquery is able to run queries from MySQL databases. This is useful when you want to import/export data from/to a MySQL database. Or when you want to join an API with a MySQL database.

## Connection

To connect a MySQL database to Anyquery, you need to provide the connection string. The connection string is a URL that contains the necessary information to connect to the database. The connection string has the following format:

```txt
username:password@protocol(address)/dbname?key1=value1&key2=value2
```

For example: `root:password&tcp(localhost:3306)/mydb?tls=true`. Refer to the [DSN documentation](https://github.com/go-sql-driver/mysql?tab=readme-ov-file#dsn-data-source-name) for more information.

To add a MySQL connection, use the following command:

```bash
anyquery connection add
```

The CLI will ask you for a connection name, the type of the connection (mysql), and the connection string. You can also provide a filter to import only the tables you need.

- The connection name is used to reference the connection in the queries. For example, if you name the connection `mysql`, you can reference the tables and views from the connection with `mysql.my_table`.
- The connection string is the URL to the MySQL database. See above for the format.
- The filter is a CEL script that filters the tables and views to import. For example, to import only the tables that start with `my_`, you can use the following filter: `table.name.startsWith("my_")`. If you don't want to filter the tables, you can leave this field empty. Refer to the [CEL syntax documentation](cel-script) for more information.

Press enter to add the connection. On the next startup, Anyquery will fetch the list of tables and views from the database and import them automatically.

Congratulations 🎉! You can now run queries on the tables and views from the MySQL database. The table name will follow this format `connection_name.[schema_name]_table_name` (e.g. `mysql.information_schema_tables`)

```sql
-- Join a table from MySQL with a table from an API
SELECT * 
FROM mysql.my_table m
JOIN github_stargazers_from_repository('julien040/anyquery') g
ON m.id = g.login

-- Insert data into a MySQL table
BEGIN;
INSERT INTO mysql.my_table (id, name) VALUES (1, 'John Doe');

-- Update data in a MySQL table
UPDATE mysql.my_table SET name = 'Jane Doe' WHERE id = 1;

-- Delete data from a MySQL table
DELETE FROM mysql.my_table WHERE id = 1;
COMMIT;

-- List all tables of the connection
SELECT * FROM mysql.information_schema_tables;
```

## Additional information

### Types

Anyquery only supports `TEXT`, `INT`, `DOUBLE`, and `BLOB` types. Due to this limitation, Anyquery automatically converts the MySQL types to the supported types.

| MySQL type           | Anyquery type | Additional information                  |
| -------------------- | ------------- | --------------------------------------- |
| `TINYINT`            | `INT`         |                                         |
| `SMALLINT`           | `INT`         |                                         |
| `MEDIUMINT`          | `INT`         |                                         |
| `INT`                | `INT`         |                                         |
| `BIGINT`             | `INT`         |                                         |
| `BIT`                | `INT`         | Use the integer value for INSERT/UPDATE |
| `FLOAT`              | `DOUBLE`      |                                         |
| `DOUBLE`             | `DOUBLE`      |                                         |
| `DECIMAL`            | `DOUBLE`      |                                         |
| `NUMERIC`            | `DOUBLE`      |                                         |
| `DATE`               | `TEXT`        |                                         |
| `TIME`               | `TEXT`        |                                         |
| `YEAR`               | `INT`         | Converted to `INT`                      |
| `DATETIME`           | `TEXT`        | Converted to RFC3339                    |
| `TIMESTAMP`          | `TEXT`        |                                         |
| `CHAR`               | `TEXT`        |                                         |
| `VARCHAR`            | `TEXT`        |                                         |
| `TINYTEXT`           | `TEXT`        |                                         |
| `TEXT`               | `TEXT`        |                                         |
| `MEDIUMTEXT`         | `TEXT`        |                                         |
| `LONGTEXT`           | `TEXT`        |                                         |
| `BINARY`             | `BLOB`        |                                         |
| `VARBINARY`          | `BLOB`        |                                         |
| `TINYBLOB`           | `BLOB`        |                                         |
| `BLOB`               | `BLOB`        |                                         |
| `MEDIUMBLOB`         | `BLOB`        |                                         |
| `LONGBLOB`           | `BLOB`        |                                         |
| `ENUM`               | `TEXT`        |                                         |
| `SET`                | `TEXT`        |                                         |
| `JSON`               | `TEXT`        |                                         |
| `GEOMETRY`           | `TEXT`        | Converted to WKT <br> Not filterable    |
| `POINT`              | `TEXT`        | Same as `GEOMETRY`                      |
| `LINESTRING`         | `TEXT`        | Same as `GEOMETRY`                      |
| `POLYGON`            | `TEXT`        | Same as `GEOMETRY`                      |
| `MULTIPOINT`         | `TEXT`        | Same as `GEOMETRY`                      |
| `MULTILINESTRING`    | `TEXT`        | Same as `GEOMETRY`                      |
| `MULTIPOLYGON`       | `TEXT`        | Same as `GEOMETRY`                      |
| `GEOMETRYCOLLECTION` | `TEXT`        | Same as `GEOMETRY`                      |

To handle `JSON` types, Anyquery converts the JSON to a string. You can then use the [many JSON functions](https://www.sqlite.org/json1.html) available in SQLite. For example:

```sql
SELECT column ->> '$.key' FROM mysql.my_table;
SELECT json_array_length(column) FROM mysql.my_table;
```

Geometry types are handled as strings with their WKT ([Well-Known Text](https://en.wikipedia.org/wiki/Well-known_text_representation_of_geometry)) representation. It's not possible to filter geometries using the WKT representation in a SELECT query. However, you can insert/update geometries using the WKT representation.
To overcome this limitation, you can create a view in MySQL, and query it in Anyquery.

```sql
INSERT INTO mysql.my_table (point_col, line_col, polygon_col, multipoint_col, multilinestring_col, multipolygon_col, geometrycollection_col) VALUES (
    'POINT(1 1)',
    'LINESTRING(0 0, 1 1)',
    'POLYGON((0 0, 1 0, 1 1, 0 1, 0 0))',
    'MULTIPOINT(0 0, 1 1)',
    'MULTILINESTRING((0 0, 1 1), (2 2, 3 3))',
    'MULTIPOLYGON(((0 0, 1 0, 1 1, 0 1, 0 0)), ((2 2, 3 2, 3 3, 2 3, 2 2)))',
    'GEOMETRYCOLLECTION(POINT(1 1), LINESTRING(0 0, 1 1), POLYGON((0 0, 1 0, 1 1, 0 1, 0 0)))'
);
```

### Functions

When querying a MySQL database, functions aren't passed to the MySQL server. Instead, they are executed in the SQLite database. It might often result in a performance hit. To avoid this, you can create a view in MySQL that computes the function and then query the view in Anyquery.

### Transactions support

We'll not claim that Anyquery is [ACID](https://en.wikipedia.org/wiki/ACID?useskin=vector) compliant. However, Anyquery supports transactions. You can start a transaction with `BEGIN;`, commit it with `COMMIT;`, or rollback it with `ROLLBACK;`.

```sql
BEGIN;
INSERT INTO mysql.my_table (id, name) VALUES (1, 'John Doe');
UPDATE mysql.my_table SET name = 'Jane Doe' WHERE id = 1;
ROLLBACK;

-- No data is inserted or updated
```

### Connection pooling

Anyquery uses a connection pool to connect to the MySQL database. We try to use as little connections as possible to avoid exhausting the MySQL server. However, note that any started transaction consumes a connection until it's committed or rolled back.
A connection might be reused between several tables or views. It's not possible to control which connection is used for a specific table or view.

### Composite primary keys

Due to limitations of SQLite (the underlying query engine), we're unable to provide INSERT/UPDATE/DELETE support for tables with composite primary keys. However, you can still query the table using SELECT queries. If this an issue for you, feel free to open an issue on the GitHub repository.

### MySQL wire-compatible databases

Anyquery is able to run queries from MySQL wire-compatible databases, such as TiDB or SingleStore. If you encounter any issues with these databases, feel free to open an issue on the GitHub repository. I cannot guarantee that the issue will be fixed, but I'll do my best to help you.
