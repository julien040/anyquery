# File plugin

Search and list files in a directory with SQL.

## Installation

```bash
anyquery install file
```

## Usage

```sql
SELECT * FROM file_list('/path/to/directory');
SELECT * FROM file_search('*.js');
```

It can be used as a basic `find` command.

```sql
-- find -name '*.ext'
SELECT * FROM file_search('*.ext');
-- find -daystart -mtime -7
SELECT * FROM file_search('*') where last_modified > datetime('now', '-7 days');
```

## Tables

### `file_list`

List files in a directory in a breadth-first order.

You can set a LIMIT so that the exploration function does not go too deep.

| Column index | Column name   | type    |
| ------------ | ------------- | ------- |
| 0            | path          | TEXT    |
| 1            | file_name     | TEXT    |
| 2            | file_type     | TEXT    |
| 3            | size          | INTEGER |
| 4            | last_modified | INTEGER |
| 5            | is_directory  | INTEGER |

### `file_search`

| Column index | Column name   | type    |
| ------------ | ------------- | ------- |
| 0            | path          | TEXT    |
| 1            | file_name     | TEXT    |
| 2            | file_type     | TEXT    |
| 3            | size          | INTEGER |
| 4            | last_modified | INTEGER |
| 5            | is_directory  | INTEGER |

## Caveats

- The plugin does not support symbolic links.
