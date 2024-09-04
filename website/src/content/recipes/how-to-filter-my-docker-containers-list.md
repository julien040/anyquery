---
title: "How to filter my docker containers list?"
description: "Learn how to filter your Docker containers list using Anyquery SQL engine. This tutorial covers filtering by status, name, image, size, and more advanced scenarios."
---

# How to Filter My Docker Containers List

Anyquery is a SQL query engine that allows you to run SQL queries on pretty much anything, including Docker containers. This tutorial will show you how to filter your Docker containers list using Anyquery.

## Prerequisites

Before starting, ensure you have the following:

- A working installation of Anyquery. Refer to the [installation guide](https://anyquery.dev/docs/#installation).
- The Docker plugin installed (`anyquery install docker`).

## Filtering Docker Containers

Once you have the Docker plugin installed, you can run SQL queries to filter your Docker containers. Here are some common scenarios for filtering Docker containers:

### List All Containers

To list all Docker containers, you can query the `docker_containers` table:

```sql
SELECT * FROM docker_containers;
```

### Filter Containers by Status

To filter containers by their status (e.g., running, exited), you can use the `state` column:

```sql
SELECT * FROM docker_containers WHERE state = 'running';
```

### Filter Containers by Name

To filter containers by their name, you can use the `names` column. Use the `LIKE` operator to match a pattern:

```sql
SELECT * FROM docker_containers WHERE names LIKE '%my_app%';
```

### Filter Containers by Image

To filter containers by the image they are using, you can use the `image` column:

```sql
SELECT * FROM docker_containers WHERE image = 'nginx:latest';
```

### Filter Containers by Size

To filter containers by their size, you can use the `size_rw` or `size_root_fs` columns:

```sql
SELECT * FROM docker_containers WHERE size_rw > 1000000;
```

### Combine Multiple Filters

You can combine multiple filters using logical operators (AND, OR) to refine your search:

```sql
SELECT * FROM docker_containers WHERE state = 'running' AND image = 'nginx:latest';
```

## Advanced Filtering

You can also filter containers by connecting to different Docker daemons or by using more complex queries. For example:

### Connect to a Different Docker Daemon

You can specify a different Docker daemon by passing the connection string as an argument to the `docker_containers` table:

```sql
SELECT * FROM docker_containers('tcp://0.0.0.0:2375') WHERE state = 'running';
```

### Filter Containers with JSON Functions

Some columns, like `labels` and `mounts`, are in JSON format. You can use JSON functions to filter these columns. For example, to filter containers with a specific label:

```sql
SELECT * FROM docker_containers WHERE json_extract(labels, '$.environment') = 'production';
```

## Conclusion

You can now easily filter your Docker containers list using Anyquery and its SQL capabilities. This tutorial covered basic filtering scenarios, but you can explore more advanced queries and combinations to fit your needs.

For more information on the Docker plugin and its schema, refer to the [Docker plugin documentation](https://anyquery.dev/integrations/docker). If you encounter any issues, refer to the [troubleshooting guide](https://anyquery.dev/docs/usage/troubleshooting/).
