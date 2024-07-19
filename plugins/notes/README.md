# Apple notes

Query and export your notes from Apple Notes with SQL.

## Installation

```bash
anyquery install notes
```

## Usage

```sql
-- Get all your notes (will probably fail with pretty output mode due to the amount of html to print. Switch to json,csv, plain, etc.)
SELECT * FROM notes_items;

-- Get all your notes with a specific title
SELECT * FROM notes_items WHERE name = 'My note title';

-- Get all your notes that talk about a specific topic
SELECT * FROM notes_items WHERE html_body LIKE '%my topic%';

-- Get your 10 most recent notes
SELECT * FROM notes_items ORDER BY creation_date DESC LIMIT 10;

-- Get the folder with the most notes
SELECT folder, COUNT(*) AS notes_count FROM notes_items GROUP BY folder ORDER BY notes_count DESC LIMIT 1;
```

```bash
# Export all your notes to a csv file
anyquery -q "SELECT * FROM notes_items" --csv > my_notes.csv
```

## Schema

| Column index | Column name       | type |
| ------------ | ----------------- | ---- |
| 0            | id                | TEXT |
| 1            | name              | TEXT |
| 2            | creation_date     | TEXT |
| 3            | modification_date | TEXT |
| 4            | html_body         | TEXT |
| 5            | folder            | TEXT |
| 6            | account           | TEXT |

## Caveats

- The plugin is read-only, you can't yet create or modify notes.
- The plugin is not yet tested on all versions of Apple Notes, please report any issues you encounter.
- The plugin caches the notes for an hour to avoid querying too often. If you want to clear the cache, run `SELECT clear_plugin_cache('notes');`.
- The plugin is a bit slow. If fetches two notes per second on a Macbook Air M1 for non-cached notes.
