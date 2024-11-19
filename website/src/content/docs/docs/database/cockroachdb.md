---
title: CockroachDB
description: Learn how to connect a CockroachDB database to Anyquery.
---

<img src="/icons/cockroachdb.svg" alt="CockroachDB" width="128" />

Anyquery is able to run queries from PostgreSQL wire-compatible databases, such as CockroachDB. This is useful when you want to import/export data from/to a CockroachDB database. Or when you want to join an API with a CockroachDB database.

## Connection

To connect a CockroachDB database to Anyquery, you need to provide the connection string. It has the following format:

```txt
postgresql://user:password@host:port/database
```

If you're using CockroachDB Cloud, you can find the connection string by clicking on the `Connect` button in the CockroachDB Cloud console. You can then copy the connection string from the modal.

Then, create a new connection using the following command:

```bash
anyquery connection add
```

Refer to the [PostgreSQL guide](../postgresql) for more information about the different parameters.
