---
title: SQL join between APIs
description: Learn how to join data from different APIs using SQL
---

Anyquery is a SQL query engine that allows you to run SQL queries on pretty much anything. One of its strengths is the ability to join data from different APIs using SQL.
In this guide, we will show you how to join data from different APIs using SQL.

## Introduction

Anyquery uses SQLite as its query engine. The implementation of joins in [SQLite](https://www.sqlite.org/optoverview.html#joins) is done with [nested loop joins](https://en.wikipedia.org/wiki/Nested_loop_join).

Due to the nature of the nested loop joins, some tables that require a parameter (e.g. `github_stars_from_user`) can't be joined directly with other tables USING the `JOIN` clause. They fail with the error `constraint failed`. Due to rate-limiting, you might also want to avoid joining tables that require a parameter with a large table.

A a work-around, we can play with `CTE` (Common Table Expressions) and `subqueries` to join the tables.

## Joining tables

To join tables, we will use the `WITH` clause to create the left table of the join and the `SELECT` statement to create the right table of the join. As an example, we will join the tables `github_my_issues` and `github_comments_from_issue`.

```sql
WITH left_table AS (
    SELECT
        *
    FROM
        github_my_issues
    LIMIT 10
)
SELECT
    *
FROM
    left_table,
    github_comments_from_issue (
        left_table.repository,
        left_table.number);
```

To speed up the query, try to put as much as possible conditions in the `WITH` clause. In this example, we put the `LIMIT 10` in the `WITH` clause to limit the number of rows in the left table.
