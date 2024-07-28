---
title: Exporting a SQL query
description: Learn how to export a SQL query to JSON, CSV, HTML, etc.
---

## TL;DR

<details>
<summary>How to export a query?</summary>

**Shell mode**

```sql
.format json
.format csv
.format plain
.format html
```

**Flag argument**

```bash
anyquery -q "SELECT * FROM table" --json
anyquery -q "SELECT * FROM table" --csv
anyquery -q "SELECT * FROM table" --plain
anyquery -q "SELECT * FROM table" --format html
```

</details>

## Introduction

Anyquery allows you to export the result of a query to various formats. See below for the full list of supported formats.

## Specify the format

### Shell mode

To specify the format once you have entered the shell mode, run:

```sql
.format <format>
```

You can also use the short version for some formats:

```sql
.json
.csv
```

To revert back to the pretty format (the default one), run:

```sql
.format pretty
-- or
.mode pretty
```

### Flag argument

To specify the format as a flag argument, run:

```bash
# JSON
anyquery -q "SELECT * FROM table" --json
# CSV
anyquery -q "SELECT * FROM table" --csv
# Plain text
anyquery -q "SELECT * FROM table" --plain
# HTML
anyquery -q "SELECT * FROM table" --format html
```

## Redirecting the output

You can redirect the output to a file using the `>` operator. For example:

```bash
anyquery -q "SELECT * FROM table" --json > output.json
```

In shell mode, you can use `.output` to specify the output file:

```sql
.output output.json
SELECT * FROM table;
```

All the next queries will be written to the `output.json` file. To revert back to the standard output, run:

```sql
.output
```

## Supported formats

### CSV

Export the result of a query to a CSV ([RFC 4180](https://www.rfc-editor.org/rfc/rfc4180.html)) file.

```sql
.format csv
-- or
.csv
```

```bash
anyquery -q "SELECT * FROM table" --csv
```

**Example**

```ansi title="Exporting the top three most starred repositories from simonw to CSV"
[0;95manyquery> [0m.csv[0m
[0;38:5:35;1mOutput mode set to CSV[0m[0m
[0m
[0;95manyquery> [0mSELECT name, stargazers_count as stars FROM github_repositories_from_user('simonw') ORDER BY stars DESC LIMIT 3;[0m
name,stars[0m
datasette,9133[0m
llm,3556[0m
shot-scraper,1591[0m
```

### HTML

Export the result of a query as an HTML table.

```sql
.format html
```

```bash
anyquery -q "SELECT * FROM table" --format html
```

**Example**

```ansi title="Exporting the top three most starred repositories from simonw to HTML"
[0;95manyquery> [0m.mode html[0m
[0;38:5:35;1mOutput mode set to html[0m[0m
[0m
[0;95manyquery> [0mSELECT name, stargazers_count as stars FROM github_repositories_from_user('simonw') ORDER BY stars DESC LIMIT 3;[0m
<table>[0m
    <thead>[0m
    <tr>[0m
        <th>name</th>[0m
        <th>stars</th>[0m
    </tr>[0m
    </thead>[0m
    <tbody>[0m
    <tr>[0m
        <td>datasette</td>[0m
        <td>9133</td>[0m
    </tr>[0m
    <tr>[0m
        <td>llm</td>[0m
        <td>3556</td>[0m
    </tr>[0m
    <tr>[0m
        <td>shot-scraper</td>[0m
        <td>1591</td>[0m
    </tr>[0m
	</tbody>[0m
</table>[0m
```

### JSON

Export the result of a query as a JSON array of objects. Each column is a key in the object.

```sql
.format json
-- or
.json
```

```bash
anyquery -q "SELECT * FROM table" --json
```

**Example**

```ansi title="Exporting the top three most starred repositories from simonw to JSON"
[0;95manyquery> [0m.json[0m
[0;38:5:35;1mOutput mode set to JSON[0m[0m
[0m
[0;95manyquery> [0mSELECT name, stargazers_count as stars FROM github_repositories_from_user('simonw') ORDER BY stars DESC LIMIT 3;[0m
[[0m
  {[0m
    "name": "datasette",[0m
    "stars": 9133[0m
  },[0m
  {[0m
    "name": "llm",[0m
    "stars": 3556[0m
  },[0m
  {[0m
    "name": "shot-scraper",[0m
    "stars": 1591[0m
  }[0m
][0m
```

### JSONL

Export the result of a query as a JSON Lines file. Each line is a JSON object representing a row separated by a newline.

```sql
.format jsonl
```

```bash
anyquery -q "SELECT * FROM table" --format jsonl
```

**Example**

```ansi title="Exporting the top three most starred repositories from simonw to JSONL"
[0;95manyquery> [0m.mode jsonl[0m
[0;38:5:35;1mOutput mode set to jsonl[0m[0m
[0m
[0;95manyquery> [0mSELECT name, stargazers_count as stars FROM github_repositories_from_user('simonw') ORDER BY stars DESC LIMIT 3;[0m
{"name":"datasette","stars":9133}[0m
{"name":"llm","stars":3556}[0m
{"name":"shot-scraper","stars":1591}[0m
```

### Line by line

Export the result of a query where each line is a column of a row. Rows are separated by `---`.

```sql
.format linebyline
```

```bash
anyquery -q "SELECT * FROM table" --format linebyline
```

**Example**

```ansi title="Exporting the top three most starred repositories from simonw line by line"
[0;95manyquery> [0m.mode linebyline[0m
[0;38:5:35;1mOutput mode set to linebyline[0m[0m
[0m
[0;95manyquery> [0mSELECT name, stargazers_count as stars FROM github_repositories_from_user('simonw') ORDER BY stars DESC LIMIT 3;[0m
name: datasette[0m
stars: 9133[0m
---[0m
name: llm[0m
stars: 3556[0m
---[0m
name: shot-scraper[0m
stars: 1591[0m
```

### Markdown

Export the result of a query as a markdown table that can be pasted to GitHub, Notion, and pretty much any markdown editor.

```sql
.format markdown
```

```bash
anyquery -q "SELECT * FROM table" --format markdown
```

**Example**

```ansi title="Exporting the top three most starred repositories from simonw to Markdown"
[0;95manyquery> [0m.mode markdown[0m
[0;38:5:35;1mOutput mode set to markdown[0m[0m
[0m
[0;95manyquery> [0mSELECT name, stargazers_count as stars FROM github_repositories_from_user('simonw') ORDER BY stars DESC LIMIT 3;[0m
| name | stars | [0m |
| ---- | ----- |[0m
| datasette    | 9133  |[0m
| llm          | 3556  |[0m
| shot-scraper | 1591  |[0m
```

### Plain text

Export the result of a query as plain text (column values separated by a tab and rows separated by a newline). This is the default one if stdout is not a terminal.

```sql
.format plain
```

```bash
anyquery -q "SELECT * FROM table" --plain
```

**Example**

```ansi title="Exporting the top three most starred repositories from simonw to plain text"
[0;95manyquery> [0m.mode plain[0m
[0;38:5:35;1mOutput mode set to plain[0m[0m
[0m
[0;95manyquery> [0mSELECT name, stargazers_count as stars FROM github_repositories_from_user('simonw') ORDER BY stars DESC LIMIT 3;[0m
datasette	9133[0m
llm	3556[0m
shot-scraper	1591[0m
```

### Plain with headers

Similar to the plain text format, but with the column names as the first row.

```sql
.format plainheader
```

```bash
anyquery -q "SELECT * FROM table" --format plainheader
```

**Example**

```ansi title="Exporting the top three most starred repositories from simonw to plain text with headers"
[0;95manyquery> [0m.mode plainheader[0m
[0;38:5:35;1mOutput mode set to plainheader[0m[0m
[0m
[0;95manyquery> [0mSELECT name, stargazers_count as stars FROM github_repositories_from_user('simonw') ORDER BY stars DESC LIMIT 3;[0m
name	stars[0m
datasette	9133[0m
llm	3556[0m
shot-scraper	1591[0m
```

### Pretty

Export the result of a query into a nice ASCII table. This is the default one if stdout is a terminal.

```sql
.format pretty
```

```bash
anyquery -q "SELECT * FROM table" --format pretty
```

**Example**

```ansi title="Exporting the top three most starred repositories from simonw to a pretty table"
[0;95manyquery> [0m.format pretty[0m
[0;38:5:35;1mOutput mode set to pretty[0m[0m
[0m
[0;95manyquery> [0mSELECT name, stargazers_count as stars FROM github_repositories_from_user('simonw') ORDER BY stars DESC LIMIT 3;[0m
+--------------+-------+[0m
|     name     | stars |[0m
+--------------+-------+[0m
| datasette    |  9133 |[0m
| llm          |  3556 |[0m
| shot-scraper |  1591 |[0m
+--------------+-------+[0m
3 results[0m
```

### TSV

Export the result of a query as a tab-separated values file (alias to plain with headers).

```sql
.format tsv
```

```bash
anyquery -q "SELECT * FROM table" --format tsv
```

**Example**

```ansi title="Exporting the top three most starred repositories from simonw to TSV"
[0;95manyquery> [0m.format tsv[0m
[0;38:5:35;1mOutput mode set to tsv[0m[0m
[0m
[0;95manyquery> [0mSELECT name, stargazers_count as stars FROM github_repositories_from_user('simonw') ORDER BY stars DESC LIMIT 3;[0m
name	stars[0m
datasette	9133[0m
llm	3556[0m
shot-scraper	1591[0m
```

### Ugly json

Similar to the JSON format, but without any indentation.

```sql
.format uglyjson
```

```bash
anyquery -q "SELECT * FROM table" --format uglyjson
```

**Example**

```ansi title="Exporting the top three most starred repositories from simonw to ugly JSON"
[0;95manyquery> [0m.format uglyjson[0m
[0;38:5:35;1mOutput mode set to uglyjson[0m[0m
[0m
[0;95manyquery> [0mSELECT name, stargazers_count as stars FROM github_repositories_from_user('simonw') ORDER BY stars DESC LIMIT 3;[0m
[{"name":"datasette","stars":9133},{"name":"llm","stars":3556},{"name":"shot-scraper","stars":1591}][0m
```

## Missing format?

If you need a format that is not supported, please [open an issue](https://github.com/julien040/anyquery/issues/new) to request it.
