## Safari plugin

Query/insert/modify your tabs in Safari.

## Installation

```bash
anyquery install safari
```

On first launch, a popup will ask if you want your terminal to control Safari. You need to accept this for the plugin to work.

## Usage

```sql
-- List all tabs
SELECT * FROM safari_tabs;
-- List all tabs in the window with the given index
SELECT * FROM safari_tabs WHERE window_index = 1;
-- Change the URL of the tabs for the given url
UPDATE safari_tabs SET url = 'https://example.com' WHERE url = 'https://github.com/';
-- Create a new tab
INSERT INTO safari_tabs (url) VALUES ('https://example.com');
```

## Schema

| Column index | Column name  | type    |
| ------------ | ------------ | ------- |
| 0            | tab_index    | INTEGER |
| 1            | title        | TEXT    |
| 2            | url          | TEXT    |
| 3            | window_name  | TEXT    |
| 4            | window_index | INTEGER |
| 5            | visible      | INTEGER |
| 6            | uid          | INTEGER |

## Caveats

- I have only tested this on macOS Sonoma, so I'm not sure if it works on other versions.
- The plugin can't close tabs yet.