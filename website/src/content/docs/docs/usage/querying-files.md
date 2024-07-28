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

While you can query local files, you can also query remote files. You can query files from HTTP, HTTPS, S3, and GCS. The syntax is the same as querying local files.

**HTTPS**

```sql
SELECT * FROM read_json('https://example.com/file.json');
```

**S3**

The `aws_access_key_id`, `aws_access_key_secret`, `region` (optional), and `version` (optional) can be passed as query parameters. Anyquery will also attempt to read them from the environment variables and `~/.aws/config`.

```sql
SELECT * FROM read_json('s3://bucket-name/file.json?aws_access_key_id=your-access-key&aws_access_key_secret=your-secret-key&region=us-west-1');
```

**GCS**

To query a file from GCS, you need to set `GOOGLE_APPLICATION_CREDENTIALS` to the path of your service account key or `GOOGLE_CREDENTIALS` to the content of your service account key.

```sql
SELECT * FROM read_json('gs://bucket-name/file.json');
```

## Stdin

You can also query files from stdin. The syntax is the same as querying local files.

```bash title="Querying JSON from stdin"
curl https://formulae.brew.sh/api/formula.json | anyquery -q "SELECT full_name, \"desc\", license FROM read_json('stdin');"
cat file.json | anyquery -q "SELECT * FROM read_json('-');"
```

## File formats

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

`objects`:

In this case, there is only one row, and each key is a column.

```json
{
  "id": 1,
  "name": "Alice"
}
```

### CSV

To query a CSV file, you need to use the `read_csv` function. The function takes one, two, or three arguments. The first argument is the path to the CSV file. The second argument is optional and is if the first row is a header. The third argument is optional and is the delimiter.

```sql
-- Query the whole CSV file
SELECT * FROM read_csv('path/to/file.csv');
-- Query a CSV file with a header
SELECT * FROM read_csv('path/to/file.csv', header=true);
-- Query a CSV file with a header and a custom delimiter
SELECT * FROM read_csv('path/to/file.csv', header=true, delimiter=';');
```

### TSV

To query a TSV file, use the `read_csv` function with the delimiter set to `\t`.

```sql
SELECT * FROM read_csv('path/to/file.tsv', delimiter='\t');
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

### Parquet

To query a Parquet file, you need to use the `read_parquet` function. The function takes one argument which is the path to the Parquet file.

```sql
SELECT * FROM read_parquet('path/to/file.parquet');
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
