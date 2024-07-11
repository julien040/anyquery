# Homebrew plugin

This plugin allows you to run SQL queries against Homebrew casks and formulae.

## Installation

```bash
anyquery install brew
```

## Usage

```sql
SELECT count(*) FROM brew_formulae;
SELECT install_90_days FROM brew_casks WHERE name = 'iterm2';
```

## Tables

### `brew_formulae`

Query all the homebrew formulae from the main repository.

#### Schema

| Column index | Column name              | type |
| ------------ | ------------------------ | ---- |
| 0            | name                     | TEXT |
| 1            | full_name                | TEXT |
| 2            | tap                      | TEXT |
| 3            | oldnames                 | TEXT |
| 4            | aliases                  | TEXT |
| 5            | versioned_formulae       | TEXT |
| 6            | description              | TEXT |
| 7            | license                  | TEXT |
| 8            | versions                 | TEXT |
| 9            | build_dependencies       | TEXT |
| 10           | dependencies             | TEXT |
| 11           | test_dependencies        | TEXT |
| 12           | recommended_dependencies | TEXT |
| 13           | optional_dependencies    | TEXT |
| 14           | revision                 | TEXT |
| 15           | install_30_days          | TEXT |
| 16           | install_90_days          | TEXT |
| 17           | install_365_days         | TEXT |

### `brew_casks`

Query all the homebrew casks from the main repository.

#### Schema

| Column index | Column name      | type |
| ------------ | ---------------- | ---- |
| 0            | token            | TEXT |
| 1            | full_token       | TEXT |
| 2            | old_tokens       | TEXT |
| 3            | tap              | TEXT |
| 4            | name             | TEXT |
| 5            | desc             | TEXT |
| 6            | homepage         | TEXT |
| 7            | url              | TEXT |
| 8            | version          | TEXT |
| 9            | sha256           | TEXT |
| 10           | install_30_days  | TEXT |
| 11           | install_90_days  | TEXT |
| 12           | install_365_days | TEXT |

## Caveats

- The plugin caches approximately 4O MB of data for 24 hours. If you have a slow internet connection, the first query might take a while to complete.
- Because the plugin caches data, it might not be up-to-date with the latest Homebrew changes. If you need the latest data, you can run `anyquery -q "select clear_plugin_cache('brew')"`.
