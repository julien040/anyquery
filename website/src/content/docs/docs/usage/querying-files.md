---
title: Querying files
description: Learn how to run SQL query on JSON, CSV, Parquet, YAML, and TOML files
---

## TL;DR

<details>
<summary>How to query a file?</summary>

:::warning
This feature is only available in the shell mode.
If you want similar features in the MySQL server, you need to create a vtable.

**JSON**

Run `SELECT * FROM read_json('path/to/file.json')` in your terminal.

**CSV**

Run `SELECT * FROM read_csv('path/to/file.csv')` in your terminal.

**Parquet**

Run `SELECT * FROM read_parquet('path/to/file.parquet')` in your terminal.

**YAML**

Run `SELECT * FROM read_yaml('path/to/file.yaml')` in your terminal.

**TOML**

Run `SELECT * FROM read_toml('path/to/file.toml')` in your terminal.

</details>

## Introduction

Anyquery is able to run SQL queries on JSON, CSV, Parquet, YAML, and TOML files. The shell mode provides syntactic sugar to query these files. In the MySQL server, you need to create a vtable to query these files which is explained [here](#mysql-server).

```sql title="Listing all the packages from Homebrew"
SELECT full_name FROM read_json('https://formulae.brew.sh/api/formula.json');
```

## Remote files

While you can query local files, you can also query remote files over HTTP and HTTPS. You can even paste a share link from GitHub, GitHub Gist, GitLab, Codeberg, Hugging Face, Google Sheets, or Dropbox, and Anyquery turns it into the raw file URL for you. The syntax is the same as querying local files.

**HTTPS**

```sql
SELECT * FROM read_json('https://example.com/file.json');
```

**Object storage (S3/GCS)**

`s3://` and `gcs://` URLs are no longer supported. To query an object in a private bucket, generate a presigned (S3) or signed (GCS) URL and query it like any other HTTPS file. Public objects work with their plain HTTPS URL.

```bash title="Presign an S3 object for 15 minutes"
aws s3 presign s3://bucket-name/file.json --expires-in 900
```

```sql
SELECT * FROM read_json('https://bucket-name.s3.amazonaws.com/file.json?X-Amz-Algorithm=…&X-Amz-Credential=…&X-Amz-Signature=…');
```

**GitHub files and gists**

A `github.com` file URL (`/blob/` or `/raw/`) is rewritten to its raw-content host; a `gist.github.com` URL is rewritten to the gist's raw form. A `/tree/` (directory) URL or a bare repo URL is rejected: name the file explicitly.

```sql
SELECT * FROM read_csv('https://github.com/owner/repo/blob/main/data.csv');
SELECT * FROM read_csv('https://gist.github.com/owner/6cad326836d38bd3a7ae');
```

**GitLab and Codeberg**

Both forges work like GitHub: paste the file's web URL and Anyquery fetches the raw one. On GitLab, a `/-/blob/` URL (subgroups included) is rewritten to its `/-/raw/` form; on Codeberg, a `/src/branch/` URL (as well as `/src/tag/` and `/src/commit/`) is rewritten to its `/raw/…` form. A URL that already points at the raw endpoint is left as is, and a directory URL (GitLab's `/-/tree/`, or a Codeberg branch URL with no file after the ref) is rejected: name the file explicitly.

```sql
SELECT * FROM read_csv('https://gitlab.com/group/subgroup/project/-/blob/main/data.csv');
SELECT * FROM read_csv('https://codeberg.org/owner/repo/src/branch/main/data.csv');
```

**Hugging Face**

Both the `hf://datasets/…` shorthand and a `huggingface.co/…/blob/…` URL are supported for dataset files; `hf://spaces/…` and `hf://models/…` are not. Globs (`*`, `?`) are not supported: name a single file.

```sql
SELECT * FROM read_csv('hf://datasets/datasets-examples/doc-formats-csv-1/data.csv');
SELECT * FROM read_csv('hf://datasets/datasets-examples/doc-formats-csv-1@main/data.csv'); -- pin a revision
```

**Google Sheets**

A share/edit link is rewritten to the sheet's CSV export, so pair it with `read_csv`:

```sql
SELECT * FROM read_csv('https://docs.google.com/spreadsheets/d/your-sheet-id/edit');
```

**Dropbox**

A share link (`?dl=0`/`?dl=1`, or no `dl` parameter at all) is rewritten to fetch the raw file:

```sql
SELECT * FROM read_csv('https://www.dropbox.com/s/abc123/data.csv?dl=0');
```

**Compression**

A file ending in `.gz` or `.zst` is decompressed automatically, whether it's local or remote. Archives (`.zip`, `.tar`, …) are not extracted.

All **remote** files are cached in the local filesystem to avoid downloading them multiple times. Every file reader accepts a `cache` argument (aliases: `cache_ttl`, `ttl`) that sets, in seconds, how long a downloaded file stays valid in that cache. It defaults to 86400 (24 hours), except for `read_html`, which defaults to 60 seconds. A non-numeric value is rejected when the table is created.

```sql
-- Re-download the file if the cached copy is older than 5 minutes
SELECT * FROM read_csv('https://example.com/data.csv', cache=300);
```

The argument only affects remote sources: a local file path always reads the file directly and never touches the cache. To clear the cache, you can use the `clear_file_cache` function.

```sql
SELECT clear_file_cache();
```

**Size limit**

A 32 GiB cap applies to three things: a remote download, the content decompressed out of a `.gz` or `.zst` (local or remote), and data piped in on stdin. A plain local file is read where it sits, so its size is never capped.

The cap exists to stop a server that never closes the connection, or a tiny compressed file engineered to expand without end, from filling your disk. Past it, the query fails rather than truncating: you never get a partial file that looks complete.

If you legitimately need more (or want to stay well under a small disk), set `ANYQUERY_MAX_DOWNLOAD_SIZE`:

```bash title="Raise the limit for one query"
ANYQUERY_MAX_DOWNLOAD_SIZE=64GiB anyquery -q "SELECT count(*) FROM read_parquet('https://example.com/huge.parquet');"
```

The value is a whole number of bytes, optionally suffixed with a unit: `KB`, `MB`, `GB`, and `TB` are powers of 1000, while `KiB`, `MiB`, `GiB`, `TiB` (and the bare `K`, `M`, `G`, `T`) are powers of 1024. Fractions such as `1.5GB` are not accepted, so write `1500MB` instead. If the value can't be read, Anyquery logs a warning and keeps the 32 GiB default rather than running with no limit at all.

:::note
Under the [sandbox](/docs/usage/sandbox), remote fetching (including all of the hosts above) requires `--allow-remote`, and stdin (`'stdin'`, `'-'`, `/dev/stdin`) is always denied regardless of `--allow-remote`.
:::

## Stdin

You can also query files from stdin. The syntax is the same as querying local files.

```bash title="Querying JSON from stdin"
curl https://formulae.brew.sh/api/formula.json | anyquery -q "SELECT full_name, \"desc\", license FROM read_json('stdin');"
cat file.json | anyquery -q "SELECT * FROM read_json('-');"
```

:::note[Bigger than memory files]
Anyquery will read the whole file in memory before parsing it. Reading from stdin will not be suitable for streaming files or files bigger than memory. In this case, save the pipe content to a file and query it.
:::

## File formats

### Any file

If you don't want to name the reader, use `read_file`. It picks the reader from the file extension and forwards every other argument to it, so `read_file('data.csv', header=true)` behaves exactly like `read_csv('data.csv', header=true)`. Supported extensions are `csv`, `tsv`, `json`, `jsonl`, `ndjson`, `parquet`, `pq`, `toml`, `yaml`, `yml`, `html` and `htm`. A trailing `.gz`, `.zst` or `.zstd` is ignored when looking at the extension, so `data.csv.gz` is read as a CSV file.

```sql
-- The extension picks the reader (.csv -> CSV, .yaml -> YAML, ...)
SELECT * FROM read_file('path/to/file.csv', header=true);
```

When the file has no extension, has a misleading one, or comes from stdin, pass `format` (`type` is an alias) to choose the reader yourself:

```sql
SELECT * FROM read_file('path/to/export', format='json');
```

### JSON

To query a JSON file, you need to use the `read_json` function. The function takes one or two arguments. The first argument is the path to the JSON file. The second argument is optional and is the JSON path to the data you want to query.

```sql
-- Query the whole JSON file
SELECT * FROM read_json('path/to/file.json');
-- Query a specific path in the JSON file
SELECT * FROM read_json('path/to/file.json', '$.items[*]');
```

You can also specify the parameters with named arguments.

```sql
SELECT * FROM read_json(path='path/to/file.json', json_path='$.items[*]');
```

#### Shapes supported

The following shapes are supported:

`records`:

```json
[
  {"id": 1, "name": "Alice"},
  {"id": 2, "name": "Bob"}
]
```

`columns`:

```json
{
  "id": [1, 2],
  "name": ["Alice", "Bob"]
}
```

`object`:

In this case, there is only one row, and each key is a column.

```json
{
  "id": 1,
  "name": "Alice"
}
```

### CSV

To query a CSV file, use the `read_csv` function. Most of the time, the path is the only argument you need: the delimiter, the header row, and the column types are detected automatically (see [auto-detection](#auto-detection) below).

```sql
-- Local file
SELECT * FROM read_csv('path/to/file.csv');
-- Remote file (cached, see the Remote files section)
SELECT * FROM read_csv('https://csvbase.com/meripaterson/stock-exchanges.csv');
-- Compressed file: .gz and .zst are decompressed on the fly
SELECT * FROM read_csv('path/to/file.csv.gz');
```

When you'd rather be explicit, pass `header` and `delimiter` (`separator` is an alias) yourself:

```sql
SELECT * FROM read_csv('path/to/file.csv', header=true, delimiter=';');
```

You can also specify a schema for the CSV file, and any query will make the best effort to parse the file according to the schema.

```sql title="Querying a CSV file with a schema" "schema='CREATE TABLE csv (id int, date varchar, cases int)'"
SELECT * FROM read_csv('https://csvbase.com/rmirror/us-covid-cases', schema='CREATE TABLE csv (id int, date varchar, cases int)', header=true) LIMIT 30;
```

:::caution
Types must be specified in their MySQL format. For example, `int`, `varchar`, `float`, etc.
:::

#### Auto-detection

When you don't say otherwise, `read_csv` looks at the beginning of the file and figures out:

- **The delimiter**: `,`, tab, `;` or `|`, whichever splits the file into consistent columns.
- **The header**: the first row becomes the column names when it looks like labels sitting on top of data (text above numbers). Otherwise columns are named `col0`, `col1`, ...
- **The column types**: a column of integers is `INTEGER`, of decimals `REAL`, of booleans `INTEGER`, and anything else is `TEXT`. Empty cells don't change a column's type.

```sql
-- Semicolon separated, with a header: nothing to pass.
SELECT name, age FROM read_csv('people.csv') WHERE age > 30;
```

Good to know:

- A file whose columns are **all text** keeps `col0`, `col1`, ...: nothing distinguishes its first row from the others. Pass `header=true` for those.
- A header row made of **numeric-looking labels** (`country,2020,2021`) is read as data for the same reason. Pass `header=true`.
- Types are detected from the beginning of the file. A value further down that doesn't match the detected type is returned as `NULL`. Pass `schema=` to set the types yourself when a column is not uniform.

Each argument turns off the part of the detection it covers, and is never second-guessed:

| Argument | Effect |
| --- | --- |
| `separator=';'` (or `separator=tab`) | the delimiter is used as given, no delimiter detection |
| `header=true` / `header=false` | the header is taken as told, in either direction |
| `schema='CREATE TABLE t(...)'` | names and types come from the schema, nothing is detected |

```sql
-- Turn all detection off
SELECT * FROM read_csv('people.csv', header=false, separator=',');
```

### TSV

A TSV file is a CSV file with tabs, and the tab delimiter is one of the ones `read_csv` detects by itself. `read_file` also picks the right reader from the `.tsv` extension.

```sql
SELECT * FROM read_csv('path/to/file.tsv');
SELECT * FROM read_file('path/to/file.tsv');
```

### HTML

You can query HTML tables using the `read_html` function. The function takes two arguments. The first argument is the URL of the HTML page. The second argument is the selector of the table.

:::note
The `read_html` function is similar to curl. It will fetch the page and extract the table using the selector. No JS is executed, and some websites block this kind of requests not coming from a browser.
:::

```sql title="Analyzing disk prices using SQL"
SELECT * FROM read_html('https://diskprices.com', '#diskprices');
```

If the CSS selector points to an element that is not a table, it will return all elements that match the selector.

```sql title="Extracting all "th" elements from the page"
anyquery> SELECT * FROM read_html('https://diskprices.com', 'th');
+----------+----------------+----------------------------------------------------+
| tag_name |    content     |                     attributes                     |
+----------+----------------+----------------------------------------------------+
| th       | Price per GB   | [{"Namespace":"","Key":"class","Val":"price-per-gb |
|          |                |  hidden"}]                                         |
| th       | Price per TB   | [{"Namespace":"","Key":"class","Val":"price-per-tb |
|          |                | "}]                                                |
| th       | Price          | null                                               |
| th       | Capacity       | null                                               |
| th       | Warranty       | null                                               |
| th       | Form Factor    | null                                               |
| th       | Technology     | null                                               |
| th       | Condition      | null                                               |
| th       | Affiliate Link | null                                               |
+----------+----------------+----------------------------------------------------+
9 results
```

To extract an attribute from an element, you can use the `->>` operator to access the JSON object's attribute.

```sql title="Extracting all links mentionning Amazon on the page"
SELECT
  value ->> 'Val' as link
FROM
  read_html ('https://diskprices.com', 'a'),
  json_each (attributes)
WHERE
  value ->> 'Key' = 'href'
  AND value ->> 'Val' LIKE '%amazon%';
```

You can also pass the parameters `cache`, `cache_ttl`, or `ttl` to the `read_html` function to reduce the time-to-live of the cache. Specify the time in seconds enclosed in single quotes.

By default, the cache is set to 60 seconds for the `read_html` table. This means that if a query is run within 60 seconds of the first query with the same URL, the result will be fetched from the cache and not from the actual website.

```sql title="Querying the GDP of countries and caching the result for 1 hour"
SELECT * FROM read_html('https://en.wikipedia.org/wiki/List_of_countries_by_GDP_(nominal)', '.wikitable', cache='3600');
```

`cache`/`cache_ttl`/`ttl` only affects a **remote** URL; a local file path always bypasses the cache and is a no-op for these parameters.

### Parquet

To query a Parquet file, you need to use the `read_parquet` function. The function takes one argument which is the path to the Parquet file.

```sql
SELECT * FROM read_parquet('https://csvbase.com/calpaterson/english-womens-football-matches.parquet');
```

### YAML

To query a YAML file, you need to use the `read_yaml` function. The function takes one argument, which is the path to the YAML file.

```sql
SELECT * FROM read_yaml('path/to/file.yaml');
```

Each key in the YAML file represents a column. Therefore, only one row is returned. This structure is similar to the `objects` shape in JSON.

### TOML

To query a TOML file, you need to use the `read_toml` function. The function takes one argument, which is the path to the TOML file.

```sql
SELECT * FROM read_toml('path/to/file.toml');
```

Each key in the TOML file is a column. Therefore, only one row is returned. It's similar to the `objects` shape in JSON.

## MySQL server

To query files in the MySQL server, you need to create a virtual table. It's a table that points to the file. The virtual table is created using the `CREATE VIRTUAL TABLE` statement. It uses the same arguments as the shell mode.
Each table named read_*file_format* in the shell has a corresponding *file_format*_reader table in the MySQL server. *file_format*_reader tables are also available in the shell mode.

```sql title="Read a JSON file"
CREATE VIRTUAL TABLE my_json_table USING json_reader('path/to/file.json');
SELECT * FROM my_json_table;
DROP TABLE my_json_table;
```

```sql title="Read a TSV file"
CREATE VIRTUAL TABLE my_tsv_table USING csv_reader('path/to/file.tsv', separator='\t');
SELECT * FROM my_tsv_table;
DROP TABLE my_tsv_table;
```

## Limitations

- You cannot observe the schema of the file using the `PRAGMA table_info` or `DESCRIBE` statement. This is due to `anyquery` rewriting your query on the fly to create a temporary virtual table for the duration of the query. To observe the schema, you need to create a virtual table as specified in the MySQL server section.
- You cannot use the `CREATE VIEW` statement with the `read_*` functions.
