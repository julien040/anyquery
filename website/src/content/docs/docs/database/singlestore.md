---
title: SingleStore
description: Learn how to connect a SingleStore database to Anyquery.
---

<img src="/icons/singlestore.svg" alt="SingleStore" width="128" />

Anyquery is able to run queries from MySQL wire-compatible databases, such as SingleStore. This is useful when you want to import/export data from/to a SingleStore database. Or when you want to join an API with a SingleStore database.

## Connection

To connect a SingleStore database to Anyquery, you need to provide the connection string. It has the following format:

```txt
user:password@tcp(domain.svc.singlestore.com:3000)/database?tls=true
```

If you're using SingleStore Helios, you can find these arguments in the SingleStore Helios console. Go to deployments, click on `Connect` on your workspace in the graph, select `SQL IDE`, and replace each argument in the connection string with the corresponding value.

Then, create a new connection using the following command:

```bash
anyquery connection add
```

Refer to the [MySQL guide](/docs/database/mysql) for more information about the different parameters.
