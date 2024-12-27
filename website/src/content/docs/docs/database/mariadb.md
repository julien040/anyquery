---
title: MariaDB
description: Learn how to connect a MariaDB database to Anyquery.
---

<img src="/icons/mariadb.svg" alt="MariaDB" width="128" />

Anyquery is able to run queries from MySQL wire-compatible databases, such as MariaDB. This is useful when you want to import/export data from/to a MariaDB database. Or when you want to join an API with a MariaDB database.

## Connection

To connect a MariaDB database to Anyquery, you need to provide the connection string. It has the following format:

```txt
user:password@tcp(domain.svc.singlestore.com:3000)/database?tls=true
```

Then, create a new connection using the following command:

```bash
anyquery connection add
```

Refer to the [MySQL guide](/docs/database/mysql) for more information about the different parameters.

## Additional information

- Geometry types of MariaDB are not supported in Anyquery. If you need to work with geometry types, consider using MySQL or PostgreSQL.
