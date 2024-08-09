---
title: Querying log files
description: Query log files with SQL
---

Anyquery can also be used to query log files. The log file can be in any format, as long as it can be parsed by the [Grok](https://www.elastic.co/guide/en/logstash/current/plugins-filters-grok.html) filter. Each named capture in the Grok pattern will be a column in the result.

## Querying log files

To query a log file, you need to use the `read_log` function. The function takes one or two or three arguments. The first argument is the path to the log file. The second argument is optional and is the Grok pattern to parse the log line. By default, the function will use the `%{GREEDYDATA:message}` pattern. The third argument is optional and is the path to a file containing custom Grok patterns.

```sql title="log.sql"
-- Query redis log file
SELECT * FROM read_log('path/to/redis.log', '(?<process>%{WORD}:%{WORD}) %{DATA:time} \* %{GREEDYDATA:message}');
-- Query nginx log file
SELECT * FROM read_log('path/to/nginx.log', '%{IPORHOST:clientip} - %{DATA:ident} %{DATA:auth} \[%{HTTPDATE:timestamp}\] "%{WORD:verb} %{DATA:request} HTTP/%{NUMBER:httpversion}" %{NUMBER:response} %{NUMBER:bytes} "%{DATA:referrer}" "%{DATA:agent}"');
-- Passing a custom pattern file
SELECT * FROM read_log('path/to/nginx.log', '%{NGINX}', filePattern='path/to/patterns.grok');
```

Arguments can also be passed as named arguments.

```sql title="log.sql"
SELECT * FROM read_log(path='path/to/redis.log', pattern='(?<process>%{WORD}:%{WORD}) %{DATA:time} \* %{GREEDYDATA:message}', filePattern='path/to/patterns.grok');
```

The table `read_log` acts exactly like other file tables (JSON, CSV, etc.). It means you can't directly use it with MySQL, it can read from remote files and stdin, you can't use `DESCRIBE` and you can't use them in views. Check the [file documentation](/docs/usage/querying-files) for more information.

## Grok patterns

Grok patterns are used to parse log lines. You can find a list of default patterns [here](https://raw.githubusercontent.com/vjeantet/grok/master/patterns/grok-patterns). You can use custom patterns by passing a file path to the `filePattern` argument. The file should contain one pattern per line. The pattern name should be in uppercase, and the pattern should be in the format `PATTERN_NAME %{PATTERN}`.

I recommend this introduction to Grok by Elastic: [Grok Basics](https://www.elastic.co/guide/en/logstash/current/plugins-filters-grok.html#_grok_basics).

Here are a few links that list some common patterns:

- [Alibaba cloud](https://www.alibabacloud.com/help/en/sls/user-guide/grok-patterns)
- [Logstash](https://github.com/logstash-plugins/logstash-patterns-core/tree/main/patterns/legacy)

### Example of custom Grok patterns

The following example shows how to use custom Grok patterns to parse a Redis log file.

```grok title="patterns.grok"
REDISTIMESTAMP %{MONTHDAY} %{MONTH} %{TIME}
REDISLOG \[%{POSINT:pid}\] %{REDISTIMESTAMP:timestamp} \* 
REDISMONLOG %{NUMBER:timestamp} \[%{INT:database} %{IP:client}:%{NUMBER:port}\] "%{WORD:command}"\s?%{GREEDYDATA:params}
```

```sql title="log.sql"
SELECT * FROM read_log('path/to/redis.log', '%{REDISLOG}', filePattern='path/to/patterns.grok');
```
