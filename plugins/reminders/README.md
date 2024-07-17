# Reminder plugin

This plugin allows you to add/modify/delete/view reminders from Apple's Reminders app with SQL
It's obviously only available on macOS.

## Setup

```bash
anyquery install reminders
```

On the first query, a popup will ask you if you want your terminal to have access to Reminders. You need to allow it.

## Usage

> ⚠️ For some reasons, the plugin is abyssmally slow with an Apple M1 on Sonoma (1 row per second). Internally, it uses AppleScript to interact with Reminders, which is not the fastest thing in the world. I feel like the Apple Script integration of Reminders is not the best.
>
> To speed up the process, I've noticed deleting completed reminders can help a lot (up to 5x faster with more than 1000 completed reminders).

```sql
-- List all reminders
SELECT * FROM reminders_items;

-- Add a reminder
INSERT INTO reminders_items (title, body, due_date) VALUES ('Buy milk', 'From the grocery store', '2024-12-31 23:59');

-- Update a reminder
UPDATE reminders_items SET title = 'Buy milk and bread' WHERE title = 'Buy milk';
UPDATE reminders_items SET completed = true WHERE title = 'Buy milk and bread';

-- Delete a reminder
DELETE FROM reminders_items WHERE title = 'Buy milk and bread';
```

## Schema

| Column index | Column name | type    |
| ------------ | ----------- | ------- |
| 0            | id          | TEXT    |
| 1            | list        | TEXT    |
| 2            | name        | TEXT    |
| 3            | body        | TEXT    |
| 4            | completed   | INTEGER |
| 5            | due_date    | TEXT    |
| 6            | priority    | INTEGER |

### Schema info

- `due_date` is a string in the format `YYYY-MM-DD HH:MM` or `YYYY-MM-DD`
- `completed` is a boolean (0 or 1)
- `priority` is an integer from 0 to 9 (0: no priority, 1–4: high, 5: medium, 6–9: low)

## Caveats

- The plugin is extremely slow with a lot of reminders.
- The plugin is not able to create new lists, only reminders in existing lists.
- The plugin is not able to create reminders with subtasks.
- The plugin does not handle sub section of reminders.
