---
title: "How to get the newest docker container?"
description: "Learn how to use Anyquery to retrieve, filter, and export the newest Docker container by querying the `docker_containers` table. Step-by-step guide included."
---

# How to Get the Newest Docker Container

Anyquery is a powerful SQL query engine that lets you run SQL queries on various data sources, including Docker containers. In this tutorial, we will show you how to get the newest Docker container using Anyquery.

## Prerequisites

Before starting, ensure you have the following:

- A working [installation of Anyquery](https://anyquery.dev/docs/#installation)
- Docker installed and running on your machine
- The Docker plugin installed: `anyquery install docker`

## Step 1: Start Anyquery

First, start Anyquery in shell mode:

```bash
anyquery
```

## Step 2: Query Docker Containers

To get the newest Docker container, we need to query the `docker_containers` table and sort the results by the `created_at` field in descending order. We will then limit the results to the first row to get the newest container.

```sql
SELECT * FROM docker_containers ORDER BY created_at DESC LIMIT 1;
```

Here is a breakdown of the query:

- `SELECT * FROM docker_containers`: Selects all columns from the `docker_containers` table.
- `ORDER BY created_at DESC`: Orders the results by the `created_at` field in descending order, so the newest container is at the top.
- `LIMIT 1`: Limits the results to just one row, which will be the newest container.

## Advanced Filtering and Exporting

You can further filter and export the data as needed.

### Filtering Containers

For example, if you only want to get the newest container that is running, you can add a WHERE clause:

```sql
SELECT * FROM docker_containers WHERE state = 'running' ORDER BY created_at DESC LIMIT 1;
```

### Exporting Results

To export the result to a CSV file, you can use the `--csv` flag:

```bash
anyquery -q "SELECT * FROM docker_containers ORDER BY created_at DESC LIMIT 1" --csv > newest_container.csv
```

Similarly, you can export the results to JSON:

```bash
anyquery -q "SELECT * FROM docker_containers ORDER BY created_at DESC LIMIT 1" --json > newest_container.json
```

### Connecting to a Remote Docker Daemon

If you need to connect to a remote Docker daemon, you can specify the connection string either in the query or as a column filter.

**Using Table Argument:**

```sql
SELECT * FROM docker_containers('tcp://0.0.0.0:2375') ORDER BY created_at DESC LIMIT 1;
```

**Using Column Filter:**

```sql
SELECT * FROM docker_containers WHERE host = 'tcp://0.0.0.0:2375' ORDER BY created_at DESC LIMIT 1;
```

## Conclusion

You have now learned how to get the newest Docker container using Anyquery. By leveraging SQL queries, you can easily filter, sort, and export Docker container data. For more advanced queries and features, refer to the [Anyquery Docker plugin documentation](https://anyquery.dev/integrations/docker).

Feel free to explore other capabilities of Anyquery and the Docker plugin to maximize your productivity!
