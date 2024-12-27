---
title: YugabyteDB
description: Learn how to connect a YugabyteDB database to Anyquery.
---

<img src="/icons/yugabyte.svg" alt="YugabyteDB" width="128" />

Anyquery is able to run queries from PostgreSQL wire-compatible databases, such as YugabyteDB. This is useful when you want to import/export data from/to a YugabyteDB database. Or when you want to join an API with a YugabyteDB database.

## Connection

To connect a YugabyteDB database to Anyquery, you need to provide the connection string. It has the following format:

```txt
postgresql://user:password@host:port/database
```

If you're using YugabyteDB Cloud, you can find the connection string by clicking on the `Connect` button in the YugabyteDB Cloud console. Select `Connect to your Application` and copy the connection string from the modal.

Then, create a new connection using the following command:

```bash
anyquery connection add
```

Refer to the [PostgreSQL guide](/docs/database/postgresql) for more information about the different parameters.
