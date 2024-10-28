---
title: Alternative languages (PRQL, PQL)
description: Use PRQL or PQL to query your data as alternative languages to SQL
---

## TL;DR

<details>
<summary>Set the language to PRQL</summary>

```bash
anyquery --prql
```

</details>
<details>
<summary>Set the language to PQL</summary>

```bash
anyquery --pql
```

</details>

## Introduction

Anyquery supports out-of-the-box alternative languages to SQL. You can use PRQL or PQL to query your data. PRQL is an attempt to make SQL more human-readable, and PQL is language similar to Microsoft Kusto Query Language (KQL).

## PRQL

[PRQL](https://prql-lang.org) allows you to write queries in a more human-readable way. FROM statement is at the beginning of the query, and the SELECT statement is at the end (which makes sense when writing a query). PRQL is available in the shell mode, stdin mode, and query as a flag argument.

```sql title="Getting the oldest stars of Jeff Delaney (from Fireship.io) that have more than 1000 stars"
from github_stars_from_user
filter stargazers_count > 1000 && user == 'codediodeio'
sort starred_at
select {
    repo_name = f"{owner}/{name}",
    starred_at,
    stargazers_count
}
take 10
```

To enable PRQL, run:

```bash
anyquery --prql
```

and install the `prqlc` CLI: [https://prql-lang.org/book/project/integrations/prqlc-cli.html#installation](https://prql-lang.org/book/project/integrations/prqlc-cli.html#installation)

Or once the shell mode is open, run:

```sql
.language prql
-- To switch back to SQL, run:
.language
```

:::tip
PRQL does not support all SQL functions that Anyquery has. To use these SQL functions, use the [S-string](https://prql-lang.org/book/reference/syntax/s-strings.html) feature. It allows you to pass direct SQL syntax to the resulting query.
:::

## PQL

[PQL](https://pql.dev) is a language that tries to bridge the gap between proprietary languages like KQL, Splunk SPL, and SQL. PQL is available in the shell mode, stdin mode, and query as a flag argument.

```sql title="Getting the oldest stars of Jeff Delaney (from Fireship.io) that have more than 1000 stars"
github_stars_from_user
| where stargazers_count > 1000 and user == 'codediodeio'
| sort by starred_at
| project repo_name = strcat(owner, '/', name), starred_at, stargazers_count
| take 10
```

To enable PQL, run:

```bash
anyquery --pql
```

Or once the shell mode is open, run:

```sql
.language pql
-- To switch back to SQL, run:
.language
```
