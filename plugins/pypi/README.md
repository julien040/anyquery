# PyPI plugin

Query with SQL the PyPI database for a package.

## Installation

```bash
anyquery install pypi
```

## Usage

```sql
SELECT * FROM pypi_versions('requests');

SELECT * FROM pypi_package('requests');
```

## Tables

### `pypi_versions`

List all versions of a package.

```sql
SELECT * FROM pypi_versions('boto3') ORDER BY created_at DESC;
```

#### Schema

| Column index | Column name    | type    |
| ------------ | -------------- | ------- |
| 0            | package_url    | TEXT    |
| 1            | package_author | TEXT    |
| 2            | version        | TEXT    |
| 3            | md5_digest     | TEXT    |
| 4            | upload_time    | TEXT    |
| 5            | filename       | TEXT    |
| 6            | version_size   | INTEGER |
| 7            | python_version | TEXT    |
| 8            | url            | TEXT    |
| 9            | yanked         | INTEGER |

### `pypi_package`

Get the package information.

```sql
SELECT * FROM pypi_package('boto3');
```

#### Schema

| Column index | Column name       | type    |
| ------------ | ----------------- | ------- |
| 0            | url               | TEXT    |
| 1            | author            | TEXT    |
| 2            | author_email      | TEXT    |
| 3            | description       | TEXT    |
| 4            | home_page         | TEXT    |
| 5            | keywords          | TEXT    |
| 6            | license           | TEXT    |
| 7            | maintainer        | TEXT    |
| 8            | maintainer_email  | TEXT    |
| 9            | documentation_url | TEXT    |
| 10           | source_code_url   | TEXT    |
| 11           | current_version   | TEXT    |
| 12           | version_count     | INTEGER |

## Caveats

- The plugin uses the PyPI API to fetch the data and is rate-limited.
