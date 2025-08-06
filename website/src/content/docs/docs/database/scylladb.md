---
title: ScyllaDB
description: Learn how to connect ScyllaDB tables to Anyquery.

---

<img src="/icons/scylladb.svg" alt="ScyllaDB" width="128" />

Anyquery is able to run queries from Cassandra-compatible databases, such as ScyllaDB.

## Connection

To connect a MariaDB database to Anyquery, you need to provide the connection string. It has the following format:

```txt
cassandra://user:password@host:port/database?option1=value1&option2=value2
```

Then, create a new connection using the following command:

```bash
anyquery connection add
```

Refer to the [Cassandra guide](/docs/database/cassandra) for more information about the different parameters.
