---
title: Cassandra
description: Learn how to connect Cassandra tables to Anyquery.
---
<img src="/icons/cassandra.svg" alt="Cassandra icon" width="192" height="192">

Cassandra is a distributed NoSQL database designed to handle large amounts of data across many commodity servers. It provides high availability with no single point of failure. Anyquery provides a way to connect to Cassandra tables and run queries on them.

## Getting started

To connect a Cassandra database to Anyquery, you need to provide the connection details. Open a shell, and run the following command:

```bash
anyquery connection add
```

The CLI will ask you for:

- a connection name (the schema name to reference the Cassandra database)
- the type of the connection (Cassandra)
- the connection string (the format is `cassandra://user:password@host:port`, e.g. `cassandra://myuser:mypassword@127.0.0.1:9042`). Refer to the [DSN documentation](#dsn) for more information on the connection string.
- a filter. It is a CEL script that filters the tables and views to import. For example, to import only the tables of the `cassandra` keyspace, you can use the following filter: `table.schema == 'cassandra'`. If you want to import all tables, you can leave this field empty. Refer to the [CEL syntax documentation](cel-script) for more information.

Press enter to add the connection. On the next startup, Anyquery will fetch the list of tables and views from the database and import them automatically.

Congratulations 🎉! You can now run queries on the tables and views from the Cassandra database. The table name will follow this format `connection_name.[schema_name]_table_name` (e.g. `mycassandra.information_schema_tables`)

```sql
-- Join a table from Cassandra with a table from an API
SELECT * 
FROM mycassandra.my_table m
JOIN github_stargazers_from_repository('julien040/anyquery') g
ON m.id = g.login
```

## Additional information

### DSN

The Data Source Name (DSN) for a Cassandra connection is a string that specifies the connection details. The format is `cassandra://user:password@host:port?option1=value1&option2=value2`.

The `user` and `password` are the credentials to connect to the Cassandra database. They often default to empty strings or `cassandra:cassandra`.

The `host` is the address of one of the Cassandra nodes in the cluster. You can specify multiple hosts separated by commas (e.g. `192.168.1.1:9042, 172.16.0.1:9043`).

#### Options

You can specify additional options in the DSN to configure the connection. Here are the options:

- `consistency`: The consistency level for the queries. It can be one of the following values: `ANY`, `ONE`, `TWO`, `THREE`, `QUORUM`, `LOCAL_QUORUM`, `EACH_QUORUM`, `SERIAL`, `LOCAL_SERIAL`, or `ALL`.
- `tls` or `ssl`: If set to `true`, the connection will use TLS/SSL. The default is `false`. It also enables the `tls_ca_cert`, `tls_cert`, and `tls_key` options.
- `tls_ca_cert`: The path to the CA certificate file for TLS/SSL.
- `tls_cert`: The path to the client certificate file for TLS/SSL.
- `tls_key`: The path to the client private key file for TLS/SSL.

```url
cassandra://myuser:mypassword@cassandra1.datacenter1:9042,cassandra2.datacenter1:9042?consistency=QUORUM&tls=true&tls_ca_cert=/path/to/ca.crt&tls_cert=/path/to/client.crt&tls_key=/path/to/client.key
```

### Functions

When querying a Cassandra database, functions aren't passed to Cassandra. Instead, they are executed in the SQLite database. It might often result in a performance hit.

### Performance

Cassandra is designed for high performance. While this integration tries to transfer the least amount of data possible, any query that requires a join or a WHERE invoquing a function will result in all the rows transferred to Anyquery, which can be slow.

Also, Cassandra is designed around SSTable. You can only filter by the primary key or clustering columns. Any other WHERE clause will result in a full table scan, transferring all the rows to Anyquery, which can be slow.

### Types

Anyquery is based on SQLite, which has a limited set of types. When querying a Cassandra database, the types are converted to SQLite types. Here is the mapping:

| Cassandra Type              | SQLite Type   | Additional Information           |
| --------------------------- | ------------- | -------------------------------- |
| `ascii`                     | `TEXT`        |                                  |
| `bigint`                    | `INTEGER`     |                                  |
| `blob`                      | `BLOB`        |                                  |
| `boolean`                   | `BOOLEAN`     |                                  |
| `counter`                   | `INTEGER`     |                                  |
| `date`                      | `DATE`        |                                  |
| `decimal`                   | `REAL`        |                                  |
| `double`                    | `REAL`        |                                  |
| `float`                     | `REAL`        |                                  |
| `inet`                      | `TEXT`        | 0.0.0.0 representation           |
| `int`                       | `INTEGER`     |                                  |
| `smallint`                  | `INTEGER`     |                                  |
| `text`                      | `TEXT`        |                                  |
| `timestamp`                 | `DATETIME`    |                                  |
| `timeuuid`                  | `TEXT`        | UUID representation              |
| `tinyint`                   | `INTEGER`     |                                  |
| `uuid`                      | `TEXT`        | UUID representation              |
| `varint`                    | `INTEGER`     |                                  |
| `list<type>`                | `JSON`        |                                  |
| `set<type>`                 | `JSON`        |                                  |
| `map<key_type, value_type>` | `JSON`        |                                  |
| `tuple<type1, type2, ...>`  | Not supported | The column will be ignored       |
| `udt<type>`                 | JSON          | Not supported. Might return `{}` |
| `frozen<type>`              | JSON          | Frozen is ignored, the type is used instead |
| `duration`                  | `INTEGER`     | Duration in nanoseconds. A month is 30 days. |

### Limitations

- Tuples are not supported. The column will be ignored.
- User-defined types (UDT) are not supported. The column will be filled with an empty JSON object `{}`.
- Frozen types are not supported. The type is used instead.
- Insert, update and delete queries are not supported. Only SELECT queries are supported.
