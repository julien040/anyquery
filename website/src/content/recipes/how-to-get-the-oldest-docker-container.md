---
title: "How to get the oldest docker container?"
description: "Learn how to use Anyquery's SQL engine to find the oldest Docker container by listing and sorting containers based on their creation time. Follow the step-by-step guide."
---

# How to Get the Oldest Docker Container

Anyquery is a powerful SQL query engine that allows you to query pretty much any data source, including Docker containers. This tutorial will guide you on how to use Anyquery to find the oldest Docker container.

## Prerequisites

Before starting, ensure you have the following:

- A working [installation of Anyquery](https://anyquery.dev/docs/#installation)
- The Docker plugin installed. You can install it by running:

```bash
anyquery install docker
```

## Step 1: List Docker Containers

First, let's confirm that we can list the Docker containers on your system. Run the following query to list all Docker containers:

```sql
SELECT * FROM docker_containers;
```

This query will return a list of all Docker containers, including their `id`, `names`, `image`, `created_at`, and other details.

## Step 2: Get the Oldest Docker Container

To find the oldest Docker container, we need to sort the containers by their `created_at` timestamp in ascending order and limit the result to the first row. Here is the SQL query to achieve this:

```sql
SELECT * FROM docker_containers ORDER BY created_at ASC LIMIT 1;
```

This query will return the oldest Docker container based on the creation time.

## Example

Here is how you can run the query using Anyquery from the command line:

```bash
anyquery -q "SELECT * FROM docker_containers ORDER BY created_at ASC LIMIT 1;"
```

This command will output the details of the oldest Docker container.

## Full SQL Query

```sql
-- List all Docker containers
SELECT * FROM docker_containers;

-- Get the oldest Docker container
SELECT * FROM docker_containers ORDER BY created_at ASC LIMIT 1;
```

## Conclusion

You have successfully used Anyquery to find the oldest Docker container on your system. You can now explore more data and perform additional queries to manage your Docker containers efficiently. For more information, refer to the [Docker plugin documentation](https://anyquery.dev/integrations/docker).
