---
title: "How to list the biggest docker images?"
description: "Learn to list the biggest Docker images using Anyquery. Follow a step-by-step guide to run SQL queries, sort images by size, and filter based on repository tags."
---

# How to List the Biggest Docker Images

In this tutorial, we will explore how to list the biggest Docker images using Anyquery. Anyquery allows you to run SQL queries on various data sources, including Docker.

## Prerequisites

Before starting, ensure you have the following:
- A working [installation of Anyquery](https://anyquery.dev/docs/#installation)
- The Docker plugin installed: `anyquery install docker`
- Docker installed and running on your machine

For more information, visit the [Docker plugin documentation](https://anyquery.dev/integrations/docker).

## Step-by-Step Guide

### Step 1: Start Anyquery

First, start the Anyquery shell:

```bash
anyquery
```

### Step 2: List Docker Images

To list Docker images, use the `docker_images` table. This table provides detailed information about all Docker images available on your machine.

### Step 3: Query for the Biggest Docker Images

To find the biggest Docker images, you can use SQL to order the images by their size in descending order. The `size` column contains the size of the Docker images in bytes.

Here is the SQL query to list the biggest Docker images:

```sql
SELECT
    repo_tags,
    size / (1024 * 1024) AS size_in_mb
FROM
    docker_images 
ORDER BY
    size DESC
LIMIT 10;
```

This query will list the 10 biggest Docker images, displaying their repository tags and sizes in megabytes.

### Example Output

```plaintext
+--------------------------+-------------+
| repo_tags                | size_in_mb  |
+--------------------------+-------------+
| my_image:latest          | 1200.55     |
| another_image:1.0.0      | 1150.34     |
| example_image:2.0        | 1020.67     |
| ...                      | ...         |
+--------------------------+-------------+
```

### Additional Tips

- **Filtering**: You can filter the images based on specific tags or repositories by adding a `WHERE` clause. For example:

    ```sql
    SELECT
        repo_tags,
        size / (1024 * 1024) AS size_in_mb
    FROM
        docker_images
    WHERE
        repo_tags LIKE '%my_repo%'
    ORDER BY
        size DESC
    LIMIT 10;
    ```

- **Detailed View**: You can add more columns to the `SELECT` statement to get more details about each image. For example, to include the `created_at` column:

    ```sql
    SELECT
        repo_tags,
        size / (1024 * 1024) AS size_in_mb,
        created_at
    FROM
        docker_images
    ORDER BY
        size DESC
    LIMIT 10;
    ```

### Step 4: Exit Anyquery

To exit Anyquery, type:

```sql
.exit
```

## Conclusion

You have successfully listed the biggest Docker images using Anyquery. This approach allows you to efficiently query and manage your Docker images using SQL. Feel free to explore more queries to gain deeper insights into your Docker environment.

For more information and advanced usage, refer to the [Anyquery documentation](https://anyquery.dev/docs/) and the [Docker plugin documentation](https://anyquery.dev/integrations/docker).
