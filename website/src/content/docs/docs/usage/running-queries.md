---
title: Running a Query
description: Learn how to run a query with anyquery
tableOfContents:
  minHeadingLevel: 2
  maxHeadingLevel: 4
---

## TL;DR

<details>
<summary>How to open the shell mode?</summary>

Run `anyquery` in your terminal without any arguments.

</details>

<details>
<summary>How to run a query as a flag argument?</summary>

Run `anyquery -q "SELECT * FROM table"` in your terminal.

</details>

<details>
<summary>How to run a query from stdin?</summary>

Run `echo "SELECT * FROM table" | anyquery`.

</details>

<details>
<summary>How to run a query using the MySQL server?</summary>

Run `anyquery server`. See the documentation for more information.

</details>

<details>
<summary>How to exit the shell mode?</summary>

Type `.exit` in the shell.

</details>

## Introduction

Thanks for installing `anyquery`. There are four different ways to run queries with `anyquery`:

- Shell mode
- As a flag argument
- From stdin
- Using the MySQL server

## Shell mode

The most used mode is the shell. Just run

```bash
anyquery
```

to enter the shell mode in SQL mode by default. 
![Example of running the shell](/images/docs/Hyper_labH4rXg@2x.png)
AnyQuery will prompt you for a SQL query, run it, and finally return the result until you exit. To exit the shell, type `.exit`.

You can also run initialization queries before entering the shell. For example, to create a table and insert some data, create a file `init.sql` with the following content:

```sql title="init.sql"
CREATE TABLE IF NOT EXISTS my_table (id INTEGER PRIMARY KEY, name TEXT);
INSERT INTO my_table (name) VALUES ('Alice');
INSERT INTO my_table (name) VALUES ('Bob');
```

Then run

```bash
anyquery --init init.sql
```

to run the initialisation queries before entering the shell.

### Commands

#### SQLite commands

The shell supports a subset of the SQLite shell commands. Here is a list of the supported commands:

- `.cd DIRECTORY` - Change the working directory to DIRECTORY.
- `.databases` - List the currently attached databases.
- `.help` - Show the help message.
- `.exit` - Exit the shell.
- `.indexes` - List all the indexes.
- `.mode MODE` - Change the output mode. The commonly used modes are `plain`, `pretty`, `json`, `markdown`, and `html`.
- `.json` - Alias for `.mode json`.
- `.jsonl` - Alias for `.mode jsonl`.
- `.csv` - Alias for `.mode csv`.
- `.output FILE` - Redirect the output to FILE.
- `.print STRING` - Print STRING.
- `.shell COMMAND` - Run COMMAND in the current directory using `exec`. Therefore, no shell globbing, piping, etc., is supported.
- `.tables` - List all the tables.
- `.language LANG` - Change the query language. The supported languages are `sql`, `prql`, and `pql`.
- `.prql` - Alias for `.language prql`.
- `.pql` - Alias for `.language pql`.
- `.sql` - Alias for `.language sql`.

#### MySQL commands

The shell also supports a subset of the MySQL shell commands. Here is a list of the supported commands:

- `SHOW DATABASES` - List the current databases.
- `SHOW TABLES` - List the tables in the current database.
- `SHOW CREATE TABLE table` - Show the create table statement for the table.
- `SHOW CREATE VIEW view` - Show the create view statement for the view.
- `DESCRIBE table` - Describe the table (does not work with file tables).
- `EXPLAIN table` - Alias for `DESCRIBE table`.

#### PostgreSQL commands

The shell also supports a subset of the PostgreSQL shell commands with slightly different semantics. Here is a list of the supported commands:

- `\l` - List the current databases.
- `\dt` - List the tables in all databases.
- `\d table` - Describe the table.
- `\d+ table` - Describe the table with more details.
- `\dv` - List the views in all databases.
- `\d+ view` - Describe the view with more details.
- `\di` - List the indexes in all databases.

## As a flag argument

To run a query as a one-off command without entering the shell, you can use the `-q` flag. For example, to run `SELECT * FROM table`, you can run

```bash
anyquery -q "SELECT * FROM table"
```

By default, the output is in the `pretty` mode unless the result is piped to another command. In that case, the output is in `plain` mode (i.e. tab-separated values without headers).

```bash title="Example of different output modes"
# Pretty mode because the output is not piped
(base) julien@MacBook-Air-Julien ~ % anyquery -q "SELECT owner, name FROM github_my_stars LIMIT 4"      
+-------------+--------------+
|    owner    |     name     |
+-------------+--------------+
| CloudCannon | pagefind     |
| surjithctly | astro-navbar |
| typst       | typst        |
| noodle-run  | noodle       |
+-------------+--------------+
4 results

# Plain mode because the output is piped
(base) julien@MacBook-Air-Julien ~ % anyquery -q "SELECT owner, name FROM github_my_stars LIMIT 4" | cat
CloudCannon     pagefind
surjithctly     astro-navbar
typst   typst
noodle-run      noodle
```

As specified in the reference, you can change the output mode using the `--format` flag. As a shortcut, you can use the `--json`, `--plain`, and `--csv` flags to respectively output in JSON, plain, and CSV formats.

To use a different query language, you can use the `--language` flag. The supported languages are `sql`, `prql`, and `pql`.

## From stdin

You can also run a query from stdin. As soon as anyquery encounters a semicolon, it will run the query, which is useful for streaming data. For example, to run `SELECT * FROM table`, you can run

```bash
echo "SELECT * FROM table" | anyquery
```

All flags and options specified in the previous section are also available when running a query from stdin.

## Using the MySQL server

`anyquery` can also act as a MySQL server so that you can use your favorite MySQL client to connect to it. To start the server, run

```bash
anyquery server
```

By default, the server listens on `127.0.0.1:8070`. You can change the host and port using the `--host` and `--port` flags. For example, to listen on `0.0.0.0:3306`, you can run

```bash
anyquery server --host "0.0.0.0" --port 3306
```

Still by default, it will open a new database named `anyquery.db` in the current directory. You can change the database using the `--database` flag. To open an in-memory database, pass the flag `--in-memory`. And to open in read-only mode, pass the flag `--readonly`.

```bash title="Examples of using the MySQL server"
# Open the database my_db
anyquery server --database my_db
# Open an in-memory database
anyquery server --in-memory
# Open the database my_db in read-only mode
anyquery server --database my_db --readonly
```

To connect to the server, you can use any MySQL client. For example, to connect using the `mysql` client, you can run

```bash

mysql -h "127.0.0.1" -P 8070
```
