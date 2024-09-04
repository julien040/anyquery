---
title: "How to query Nginx logs using SQL?"
description: "Learn to query Nginx logs using SQL with Anyquery. Understand `grok` patterns, filter and analyze data, and export results to formats like CSV. Full guide included."
---

# How to Query Nginx Logs Using SQL

Anyquery is a powerful SQL query engine that allows you to query a wide range of data sources, including log files. In this tutorial, we'll walk through how to query Nginx logs using SQL in Anyquery.

## Prerequisites

Before starting, ensure you have the following:
- A working installation of Anyquery. For installation instructions, see [here](https://anyquery.dev/docs/#installation).
- Access to your Nginx log file.

Anyquery uses `grok` patterns for parsing log files. Make sure you understand the format of your Nginx log file and the corresponding `grok` pattern.

## Step-by-Step Guide

### Step 1: Identify the Grok Pattern

Nginx logs are commonly formatted in a standard way. Here's an example of an Nginx log entry:

```
127.0.0.1 - - [12/Oct/2022:06:25:24 +0000] "GET /index.html HTTP/1.1" 200 612 "-" "Mozilla/5.0 (X11; Ubuntu; Linux x86_64; rv:103.0) Gecko/20100101 Firefox/103.0"
```

The corresponding `grok` pattern for this format is:

```
%{IPORHOST:clientip} - %{DATA:ident} %{DATA:auth} \[%{HTTPDATE:timestamp}\] "%{WORD:verb} %{DATA:request} HTTP/%{NUMBER:httpversion}" %{NUMBER:response} %{NUMBER:bytes} "%{DATA:referrer}" "%{DATA:agent}"
```

### Step 2: Query the Log File

Let's assume your Nginx log file is located at `/var/log/nginx/access.log`. You can use the `read_log` function to query the file. Here’s how you can do it:

```sql
SELECT * FROM read_log('/var/log/nginx/access.log', '%{IPORHOST:clientip} - %{DATA:ident} %{DATA:auth} \\[%{HTTPDATE:timestamp}\\] "%{WORD:verb} %{DATA:request} HTTP/%{NUMBER:httpversion}" %{NUMBER:response} %{NUMBER:bytes} "%{DATA:referrer}" "%{DATA:agent}"');
```

### Step 3: Filtering and Analyzing Data

You can filter and analyze the data using SQL. For example, to find the top 10 IP addresses with the most requests:

```sql
SELECT clientip, COUNT(*) as request_count
FROM read_log('/var/log/nginx/access.log', '%{IPORHOST:clientip} - %{DATA:ident} %{DATA:auth} \\[%{HTTPDATE:timestamp}\\] "%{WORD:verb} %{DATA:request} HTTP/%{NUMBER:httpversion}" %{NUMBER:response} %{NUMBER:bytes} "%{DATA:referrer}" "%{DATA:agent}"')
GROUP BY clientip
ORDER BY request_count DESC
LIMIT 10;
```

To find the number of requests per response status code:

```sql
SELECT response, COUNT(*) as request_count
FROM read_log('/var/log/nginx/access.log', '%{IPORHOST:clientip} - %{DATA:ident} %{DATA:auth} \\[%{HTTPDATE:timestamp}\\] "%{WORD:verb} %{DATA:request} HTTP/%{NUMBER:httpversion}" %{NUMBER:response} %{NUMBER:bytes} "%{DATA:referrer}" "%{DATA:agent}"')
GROUP BY response;
```

### Step 4: Exporting Results

You can export the query results to various formats like JSON, CSV, etc. For example, to export the results to a CSV file:

```bash
anyquery -q "SELECT clientip, COUNT(*) as request_count FROM read_log('/var/log/nginx/access.log', '%{IPORHOST:clientip} - %{DATA:ident} %{DATA:auth} \\[%{HTTPDATE:timestamp}\\] "%{WORD:verb} %{DATA:request} HTTP/%{NUMBER:httpversion}" %{NUMBER:response} %{NUMBER:bytes} "%{DATA:referrer}" "%{DATA:agent}"') GROUP BY clientip ORDER BY request_count DESC LIMIT 10;" --csv > nginx_top_ips.csv
```

### Conclusion

You have successfully queried Nginx logs using SQL in Anyquery. By leveraging the power of SQL, you can filter, analyze, and export log data with ease. For more information on querying logs, refer to the [log querying documentation](https://anyquery.dev/docs/usage/querying-log.md).

Happy querying!
