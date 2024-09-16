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

Grok patterns are used to parse log lines. They are needed to specify the format of the log line.
You can find a list of the default patterns [here](https://raw.githubusercontent.com/vjeantet/grok/master/patterns/grok-patterns).

You can use custom predefined patterns by passing a file path to the `filePattern` argument. The file should contain one pattern per line. The pattern name should be in uppercase, and the pattern should be in the format `PATTERN_NAME %{PATTERN}`.

But what is a Grok pattern? A Grok pattern is a regular expression with named captures. For example, the pattern `%{IP:client}` will match an IP address and name the matched value `client`. Let's say we have the following log line:

```log
[1234] 12 Dec 12:00 gibberish
```

You can use the following Grok pattern to parse it:

```grok
\[%{POSINT:pid}\] %{MONTHDAY} %{MONTH} %{TIME} %{GREEDYDATA:message}
```

Using [Grok], you can parse the log line by specifying its format. In our case, we match a positive integer embedded in square brackets by writing `\[%{POSINT:pid}\]`. The `POSINT` is a built-in pattern that matches a positive integer. We append `:pid` to name the matched value with the column name `pid`.

The `MONTHDAY`, `MONTH`, and `TIME` are built-in patterns that match the day of the month, the month, and the time, respectively. Because they don't have a name, they will be ignored. However, you're still required to add them to match the log line. Otherwise, the line will be filled with `NULL` values.

The `GREEDYDATA` pattern matches any character except for a newline. It will match the rest of the log line and name it `message`.

We should get the following result:

| pid  | message   |
| ---- | --------- |
| 1234 | gibberish |

However, `pid` is a string, and you might want to convert it to an integer. You can do this by using grok transformations.

```sql ins=":int"
SELECT pid, message FROM read_log('path/to/logfile.log', '\[%{POSINT:pid:int}\] %{MONTHDAY} %{MONTH} %{TIME} %{GREEDYDATA:message}');
```

You have to append `:int` to the pattern name to convert the matched value to an integer. Note that most of the time, this is not needed as SQLite is dynamically typed. Therefore, writing `pid+1` will automatically convert `pid` to an integer. If you want to append a string to `pid`, you can use the `||` operator.

```sql "pid || ' is a number'"
SELECT pid || ' is a number' FROM read_log('path/to/logfile.log', '\[%{POSINT:pid}\] %{MONTHDAY} %{MONTH} %{TIME} %{GREEDYDATA:message}');
```

Not let's say we want the whole timestamp in a single column. We can use the following Grok pattern:

```grok ins="(?<timestamp>%"
\[%{POSINT:pid}\] (?<timestamp>%{MONTHDAY} %{MONTH} %{TIME}) %{GREEDYDATA:message}
```

Grok is just syntaxic sugar for regular expressions. Therefore, when we want to merge several patterns into one, we can use named expressions. We use `(?<name>pattern)` to name the matched value. We can then access it by its name.

| pid  | timestamp    | message   |
| ---- | ------------ | --------- |
| 1234 | 12 Dec 12:00 | gibberish |

### Additional resources

I recommend this introduction to Grok by Elastic: [Grok Basics](https://www.elastic.co/guide/en/logstash/current/plugins-filters-grok.html#_grok_basics).

Here are a few links that list some common patterns:

- [Alibaba cloud](https://www.alibabacloud.com/help/en/sls/user-guide/grok-patterns)
- [Logstash](https://github.com/logstash-plugins/logstash-patterns-core/tree/main/patterns/legacy)

To debug Grok patterns, you can use the [Grok debugger](http://grokconstructor.appspot.com/do/match).

### Examples of custom Grok patterns

The following example shows how to use custom Grok patterns to parse a Redis log file.

```grok title="patterns.grok"
REDISTIMESTAMP %{MONTHDAY} %{MONTH} %{TIME}
REDISLOG \[%{POSINT:pid}\] %{REDISTIMESTAMP:timestamp} \* 
REDISMONLOG %{NUMBER:timestamp} \[%{INT:database} %{IP:client}:%{NUMBER:port}\] "%{WORD:command}"\s?%{GREEDYDATA:params}
```

```sql title="log.sql"
SELECT * FROM read_log('path/to/redis.log', '%{REDISLOG}', filePattern='path/to/patterns.grok');
```
