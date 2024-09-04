---
title: "How to query Kubernetes logs using SQL?"
description: "Learn to query Kubernetes logs using SQL with Anyquery, utilizing Grok patterns and the `read_log` function for structured data extraction and effective log analysis."
---

# How to Query Kubernetes Logs Using SQL

## Introduction

Anyquery is a query engine that allows you to run SQL queries on various data sources, including log files. In this tutorial, you will learn how to query Kubernetes logs using SQL with Anyquery. We'll use the `read_log` table function to parse Kubernetes logs and extract meaningful information using SQL queries.

For more information on how to install Anyquery, refer to the [installation documentation](https://anyquery.dev/docs/#installation).

## Prerequisites

Before starting, ensure you have the following:

- A working installation of Anyquery
- Access to your Kubernetes log files

If you need more information on how to install Anyquery, please refer to the [installation documentation](https://anyquery.dev/docs/#installation).

## Step 1: Understand the Log Format

Kubernetes logs are usually in a standardized format, and we will use Grok patterns to parse them. Grok is a tool used to extract unstructured data into structured data (fields).

Here is an example of a Kubernetes log entry:

```
2023-07-21T14:00:00Z stdout F This is a log message
```

This log entry consists of the following components:
1. Timestamp (`2023-07-21T14:00:00Z`)
2. Stream type (`stdout`)
3. Log level (`F` for Fatal)
4. Log message (`This is a log message`)

## Step 2: Define a Grok Pattern

To parse the Kubernetes log entry, we need to define a Grok pattern. Here is an example of a Grok pattern for the log entry:

```
%{TIMESTAMP_ISO8601:timestamp} %{WORD:stream} %{WORD:log_level} %{GREEDYDATA:message}
```

- `%{TIMESTAMP_ISO8601:timestamp}`: Matches the timestamp and stores it in the `timestamp` field.
- `%{WORD:stream}`: Matches the stream type (stdout/stderr) and stores it in the `stream` field.
- `%{WORD:log_level}`: Matches the log level and stores it in the `log_level` field.
- `%{GREEDYDATA:message}`: Matches the rest of the line as the log message and stores it in the `message` field.

## Step 3: Query the Log File

Now that we have the Grok pattern, we can use the `read_log` table function to query the log file. Here is an example SQL query:

```sql
SELECT * FROM read_log('path/to/kubernetes.log', '%{TIMESTAMP_ISO8601:timestamp} %{WORD:stream} %{WORD:log_level} %{GREEDYDATA:message}');
```

This query will parse the log file and return the structured data as a table.

### Example Queries

1. **Retrieve all logs:**

   ```sql
   SELECT * FROM read_log('path/to/kubernetes.log', '%{TIMESTAMP_ISO8601:timestamp} %{WORD:stream} %{WORD:log_level} %{GREEDYDATA:message}');
   ```

2. **Filter logs by log level:**

   ```sql
   SELECT * FROM read_log('path/to/kubernetes.log', '%{TIMESTAMP_ISO8601:timestamp} %{WORD:stream} %{WORD:log_level} %{GREEDYDATA:message}') WHERE log_level = 'F';
   ```

3. **Retrieve logs within a specific time range:**

   ```sql
   SELECT * FROM read_log('path/to/kubernetes.log', '%{TIMESTAMP_ISO8601:timestamp} %{WORD:stream} %{WORD:log_level} %{GREEDYDATA:message}')
   WHERE timestamp BETWEEN '2023-07-21T14:00:00Z' AND '2023-07-21T15:00:00Z';
   ```

4. **Count logs by stream type:**

   ```sql
   SELECT stream, COUNT(*) as count FROM read_log('path/to/kubernetes.log', '%{TIMESTAMP_ISO8601:timestamp} %{WORD:stream} %{WORD:log_level} %{GREEDYDATA:message}')
   GROUP BY stream;
   ```

## Conclusion

You have successfully learned how to query Kubernetes logs using SQL with Anyquery. By defining a Grok pattern and using the `read_log` table function, you can extract and analyze log data effectively.

For more information on Grok patterns, check out the [Grok documentation](https://www.elastic.co/guide/en/logstash/current/plugins-filters-grok.html). For more details on querying log files, refer to the [Anyquery log querying documentation](https://anyquery.dev/docs/usage/querying-log).
