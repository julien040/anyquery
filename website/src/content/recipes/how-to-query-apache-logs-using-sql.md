---
title: "How to query Apache logs using SQL?"
description: "Learn to query Apache logs with SQL using Anyquery. This guide covers identifying log formats, using Grok patterns, filtering data, and exporting results."
---

# How to Query Apache Logs Using SQL with Anyquery

Anyquery is a versatile SQL query engine that allows you to query various types of data, including log files. In this tutorial, we'll walk through the steps to query Apache logs using SQL with Anyquery.

## Introduction

Anyquery lets you run SQL queries on virtually anything. For querying log files, it uses the `read_log` table function, which can parse log entries using Grok patterns. Grok patterns are essential for matching and extracting data from log files.

Before proceeding, make sure you have Anyquery installed. If not, follow the [installation instructions](https://anyquery.dev/docs/#installation).

## Prerequisites

- A working installation of Anyquery.
- Apache log files that you want to query.

## Step 1: Identify Your Apache Log Format

Typically, Apache logs come in two formats: Common Log Format (CLF) and Combined Log Format. Here are the Grok patterns for both formats:

- **Common Log Format (CLF):**
  ```
  %{COMMONAPACHELOG}
  ```

- **Combined Log Format:**
  ```
  %{COMBINEDAPACHELOG}
  ```

## Step 2: Query Apache Logs

To query your Apache logs, you'll use the `read_log` table function. This function requires two main arguments:

1. The path to the log file.
2. The Grok pattern to parse the log entries.

### Example Queries

#### Query Common Log Format

```sql
SELECT * FROM read_log(
  'path/to/access.log', 
  '%{COMMONAPACHELOG}'
);
```

#### Query Combined Log Format

```sql
SELECT * FROM read_log(
  'path/to/access.log', 
  '%{COMBINEDAPACHELOG}'
);
```

You can also use named arguments for better readability:

```sql
SELECT * FROM read_log(
  path='path/to/access.log', 
  pattern='%{COMBINEDAPACHELOG}'
);
```

### Filtering and Displaying Specific Columns

You can filter logs and display specific columns by using standard SQL clauses. For example, to list all requests from a specific IP address:

```sql
SELECT clientip, request, status 
FROM read_log(
  'path/to/access.log', 
  '%{COMBINEDAPACHELOG}'
)
WHERE clientip = '192.168.1.1';
```

### Aggregate Data

To get the count of requests per HTTP status code:

```sql
SELECT
  status,
  COUNT(*) AS request_count
FROM read_log(
  'path/to/access.log', 
  '%{COMBINEDAPACHELOG}'
)
GROUP BY status;
```

## Step 3: Advanced Usage

### Using Custom Grok Patterns

If your log format is custom or slightly different, you might need to define custom Grok patterns. Create a pattern file (`patterns.grok`) and use it:

```grok
CUSTOMAPACHELOG %{IPORHOST:clientip} %{USER:ident} %{USER:auth} \[%{HTTPDATE:timestamp}\] "%{WORD:verb} %{DATA:request} HTTP/%{NUMBER:httpversion}" %{NUMBER:status} %{NUMBER:bytes}
```

Query using the custom pattern file:

```sql
SELECT * FROM read_log(
  path='path/to/access.log', 
  pattern='%{CUSTOMAPACHELOG}', 
  filePattern='path/to/patterns.grok'
);
```

### Exporting Results

You can export the query results to different formats like JSON, CSV, etc.:

```bash
anyquery -q "SELECT * FROM read_log('path/to/access.log', '%{COMBINEDAPACHELOG}')" --json > output.json
anyquery -q "SELECT * FROM read_log('path/to/access.log', '%{COMBINEDAPACHELOG}')" --csv > output.csv
```

## Conclusion

By using Anyquery, you can seamlessly query Apache logs using SQL. This approach leverages the flexibility of SQL to filter, aggregate, and analyze log data efficiently. For more details on the `read_log` table function and Grok patterns, refer to the [documentation](https://anyquery.dev/docs/usage/querying-log).

Happy querying!
