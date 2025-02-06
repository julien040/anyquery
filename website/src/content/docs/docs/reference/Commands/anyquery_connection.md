---
title: anyquery connection
description: Learn how to use the anyquery connection command in Anyquery.
---

Manage connections to other databases

### Synopsis

Anyquery can connect to other databases such as MySQL, PostgreSQL, SQLite, etc.
You can add, list, and delete connections.

Each connection has a name, a type, and a connection string. You can also define a small CEL script to filter which tables to import.
The connection name will be used as the schema name in the queries. 
For example, if you have a connection named "mydb", a schema named "information_schema" and a table named "tables", you can query it with "SELECT * FROM mydb.information_schema_tables".


```bash
anyquery connection [flags]
```

### Examples

```bash
# List the connections
anyquery connection list

# Add a connection
anyquery connection add mydb mysql mysql://user:password@localhost:3306/dbname "table.schema == 'public'"

# Remove a connection
anyquery connection remove mydb

```

### Options

```bash
      --csv             Output format as CSV
      --format string   Output format (pretty, json, csv, plain)
  -h, --help            help for connection
      --json            Output format as JSON
      --plain           Output format as plain text
```

### SEE ALSO

* [anyquery](../anyquery)	 - A tool to query any data source
* [anyquery connection add](../anyquery_connection_add)	 - Add a connection
* [anyquery connection list](../anyquery_connection_list)	 - List the connections
* [anyquery connection remove](../anyquery_connection_remove)	 - Remove a connection
