---
title: Data type
description: Learn about the data types supported by AnyQuery
tableOfContents:
  minHeadingLevel: 2
  maxHeadingLevel: 4
---

Because Anyquery uses SQLite as its query engine, it supports the following data types:

- TEXT
- INTEGER
- REAL
- BLOB
- NULL

But SQLite is considered a loosely typed database, so you can store any type of data in any column. For example, in a Notion database, the formula column might return a number or a string. Your queries should be able to handle this. Note that's considered as a feature, not a bug.

## Type affinity

When you create a table, the column type is not strict, it's just a suggestion. SQLite uses a concept called type affinity to determine the type of a value. SQLite might try to convert the value to the type you specified, but it's not guaranteed. See the [SQLite documentation](https://www.sqlite.org/datatype3.html) for more information.

## Other data types and replacements

### Date and time

SQLite does not have a built-in date and time data type. You can store date and time as TEXT, INTEGER, or REAL. As a convention, `anyquery` uses TEXT and the RFC3339 format for date and time. RFC3339 is mostly a subset of ISO8601, unless for some quirks. For example, valid dates in RFC3339 are `2021-01-01T00:00:00Z`, `2021-01-01T00:00:00+00:00`, and `2021-01-01 00:00:00`.

#### Modifiying the format

Anyquery ships with a `strftime` function that allows you to format the date and time. The syntax is similar to the `strftime` function in C. For example, to format the date and time as `YYYY-MM-DD`, you can run:

```sql
SELECT strftime('%Y-%m-%d', '2021-01-01T00:00:00Z');
```

#### Modifying the date

SQLite also has `date`, `datetime`, and `time` functions that format respectively to `YYYY-MM-DD`, `YYYY-MM-DD HH:MM:SS`, and `HH:MM:SS`. For example, to format the date and time as `YYYY-MM-DD`, you can run:

```sql
SELECT date('2021-01-01T00:00:00Z');
```

These dates also supports time modifiers. For example, to add 1 day to a date, you can run:

```sql
SELECT date('2021-01-01T00:00:00Z', '+1 day');
```

**Modifiers:**

- NNN days
- NNN hours
- NNN minutes
- NNN seconds
- NNN months
- NNN years
- ±HH:MM
- ±HH:MM:SS
- ±HH:MM:SS.SSS
- ±YYYY-MM-DD
- ±YYYY-MM-DD HH:MM
- ±YYYY-MM-DD HH:MM:SS
- ±YYYY-MM-DD HH:MM:SS.SSS
- ceiling
- floor
- start of month
- start of year
- start of day
- weekday N
- unixepoch
- julianday
- auto
- localtime
- utc
- subsec
- subsecond

See the [SQLite documentation](https://www.sqlite.org/lang_datefunc.html) for more information.

### Arrays

SQLite does not have a built-in array data type. Conventionally, arrays are stored as a JSON array in a TEXT column. For example, to store an array of numbers, you can run:

```sql
CREATE TABLE example (numbers TEXT);
INSERT INTO example (numbers) VALUES ('[1, 2, 3]');
```

You can later use the `->>` operator to extract the array elements. For example, to extract the first element, you can run:

```sql
SELECT json_extract(numbers, '$[0]') FROM example;
```

You can also use the `json_each` function to extract all the elements. For example, to extract all the elements, you can run:

```sql
SELECT value FROM example, json_each(numbers);
```

### Maps, dictionaries, and objects

Similarly, SQLite does not have a built-in map, dictionary, or object data type. Conventionally, maps are stored as a JSON object in a TEXT column. For example, to store a map of numbers, you can run:

```sql
CREATE TABLE example (numbers TEXT);
INSERT INTO example (numbers) VALUES ('{"one": 1, "two": 2, "three": 3}');
```

You can later use the `->>` operator to extract the map elements. For example, to extract the value of the key `one`, you can run:

```sql
SELECT json_extract(numbers, '$.one') FROM example;
```

You can also use the `json_each` function to extract all the elements. For example, to extract all the elements, you can run:

```sql
SELECT key, value FROM example, json_each(numbers);
```

### Booleans

SQLite does not have a built-in boolean data type. Conventionally, booleans are stored as 0 or 1 in an INTEGER column. For example, to store a boolean, you can run:

```sql
CREATE TABLE example (flag INTEGER);
INSERT INTO example (flag) VALUES (1);
```

However, SQLite still has `true` as an alias for 1 and `false` as an alias for 0. For example, to check if a value is true, you can run:

```sql
SELECT 1 IS true;
```

