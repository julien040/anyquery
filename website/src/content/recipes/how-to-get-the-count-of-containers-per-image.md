---
title: "How to get the count of containers per image?"
description: "Learn how to use Anyquery to get the count of Docker containers per image with a simple SQL query. Follow step-by-step instructions to set up and run your query."
---

# How to Get the Count of Containers per Image in Docker

Anyquery allows you to run SQL queries on virtually anything, including Docker containers and images. In this tutorial, we will show you how to get the count of containers per image using Anyquery.

## Prerequisites

Before starting, ensure you have the following:
- A working installation of Anyquery. Refer to the [installation guide](https://anyquery.dev/docs/#installation).
- The Docker plugin installed. Run the following command to install the plugin:

  ```bash
  anyquery install docker
  ```

## Step 1: Ensure Docker Daemon Is Running

Ensure your Docker daemon is running. If not already running, start it using the appropriate command for your system:

- For Linux:
  ```bash
  sudo systemctl start docker
  ```

- For MacOS:
  Open the Docker Desktop application.
  
- For Windows:
  Open the Docker Desktop application.

## Step 2: Launch Anyquery

Open a terminal and launch the Anyquery shell:

```bash
anyquery
```

## Step 3: Query Docker Containers

To get the count of containers per image, you can use a simple SQL query. The `docker_containers` table from the Docker plugin contains the necessary data, specifically the `image` and `id` columns.

### Query

Run the following SQL query in the Anyquery shell to get the count of containers per image:

```sql
SELECT image, COUNT(id) AS container_count
FROM docker_containers
GROUP BY image
ORDER BY container_count DESC;
```

### Explanation

- `SELECT image, COUNT(id) AS container_count`: Selects the image name and counts the number of containers (`id`) for each image.
- `FROM docker_containers`: Specifies the `docker_containers` table as the data source.
- `GROUP BY image`: Groups the results by image name.
- `ORDER BY container_count DESC`: Orders the results by the container count in descending order.

## Step 4: View Results

After running the query, you will see the count of containers per image in the terminal.

### Example Output

```sql
+-------------------------+-----------------+
|         image           | container_count |
+-------------------------+-----------------+
| alpine                  |               5 |
| nginx                   |               3 |
| redis                   |               2 |
| postgres                |               1 |
+-------------------------+-----------------+
```

This output shows the images and the corresponding number of containers running for each image.

## Conclusion

You have successfully queried the count of containers per image using Anyquery. You can now explore and analyze more data from your Docker daemon using SQL queries with Anyquery. For more detailed information on the Docker plugin, refer to the [Docker plugin documentation](https://anyquery.dev/integrations/docker).

Happy querying!
