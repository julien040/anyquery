---
title: Generate a CSV full of random data
description: Learn how to generate a CSV file full of random data using Anyquery.
---

Anyquery is a SQL query engine that enables you to execute SQL queries on virtually anything, including random data. Moreover, as it can export a query result to a CSV file, you can generate a CSV file full of random data with a straightforward SQL query.

As developers, we often need random data for testing purposes. Anyquery comes handy in such cases.

To start, install the random data plugin:

```bash
anyquery install random
```

You now have access to three tables: `random_people`, `random_password`, and `random_internet`. See the [random data plugin documentation](/integrations/random) for more information.

For example, the following query generates a CSV file full of random people:

```bash
anyquery -q "SELECT first_name, last_name, email, random_intn(40)+18 AS age FROM random_people LIMIT 100" --csv > people.csv
```

:::caution
Specify a `LIMIT` clause. Otherwise, the query will generate rows until the end of time.
:::

Additionally, you can modify each column using functions such as `upper`. For example, the following query generates a CSV file full of random people and converts the `first_name` column to uppercase:

```bash
anyquery -q "SELECT upper(first_name), last_name, email, random_intn(40)+18 AS age FROM random_people LIMIT 100" --csv > people.csv
```

See the [functions documentation](/docs/reference/functions) for more information on the available functions.
