---
title: TiDB
description: Learn how to connect a TiDB database to Anyquery.
---

<img src="/icons/tidb.svg" alt="TiDB" width="128" />

Anyquery is able to run queries from MySQL wire-compatible databases, such as TiDB. This is useful when you want to import/export data from/to a TiDB database. Or when you want to join an API with a TiDB database.

## Connection

To connect a TiDB database to Anyquery, you need to provide the connection string. It has the following format:

```txt
id.root:password@tcp(domain.aws.tidbcloud.com:4000)/database?tls=true
```

If you're using TiDB Cloud, you can find the connection string by clicking on the `Connect` button in the TiDB Cloud console. Select `Go` as the driver (in Connect With) and copy the connection string after `db, err := sql.Open("mysql",` in the Go code. Replace `tls=tidb` with `tls=true` in the connection string.

Then, create a new connection using the following command:

```bash
anyquery connection add
```

Refer to the [MySQL guide](../mysql) for more information about the different parameters.
