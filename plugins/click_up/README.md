# ClickUp

ClickUp is a project management tool that makes it easy to organize work and keep track of your team's progress. Query it using SQL with Anyquery.

## Configuration

First, install the plugin:

```bash

anyquery install clickup

```

Anyquery will ask you to create a ClickUp API token. To create one, go to your ClickUp account settings (profile icon > settings). Scroll down the left sidebar and click on "Apps".
Then copy the API token at the top of the page, and paste it into Anyquery.

## Usage

Most of the times, the table requires arguments (`list_id`, `workspace_id`, `document_id`, `space_id`, etc.) to fetch the data. You can provide these arguments as a parameter to the table function (e.g. `clickup_tasks('list_id')`). You can also use the `WHERE` clause to pass these arguments (e.g. `WHERE doc_id = 'doc_id'`).

To retrieve these IDs, follow these instructions:

- `list_id`: Go to the list you want to query, and copy the ID from the URL. For example, in `https://app.clickup.com/12345678/v/l/li/98765432`, the list ID is `98765432`.
- `workspace_id`: Go to the workspace you want to query, and copy the first part of the URL. For example, in `https://app.clickup.com/12345678/v/l/li/98765432`, the workspace ID is `12345678`.
- `document_id`: Go to the document you want to query, and copy the ID from the URL after `/dc/`. For example, in `https://app.clickup.com/12345678/v/dc/98765432/dakg-78`, the document ID is `98765432`. Omit the part after the document ID.
- `folder_id`: Go to the folder you want to query, and copy the ID from the URL. For example, in `https://app.clickup.com/12345678/v/o/f/98765432`, the folder ID is `98765432`.
- `space_id`: Go to the space you want to query, and copy the ID from the URL. For example, in `https://app.clickup.com/12345678/v/s/98765432`, the space ID is `98765432`.

The clickup hierarchy is as follows: `workspace` > `space` > `folder` > `list` > `task`. Documents are part of a workspace, and pages are part of a document. Refer to the [ClickUp documentation](https://help.clickup.com/hc/en-us/articles/13856392825367-Intro-to-the-Hierarchy) for more information.

```sql
-- List all the tasks in a list
SELECT * FROM clickup_tasks('list_id');
SELECT * FROM clickup_tasks WHERE list_id = 'list_id';

-- Count the number of tasks per status
SELECT status, COUNT(*) FROM clickup_tasks('list_id') GROUP BY status;

-- Access a custom field of a task
SELECT custom_fields ->> '$.custom_field_name' FROM clickup_tasks('list_id');

-- Search a document by name
SELECT * FROM clickup_docs('workspace_id') WHERE name LIKE '%document_name%';

-- Concat all the pages of a document
SELECT group_concat(content) FROM clickup_pages('workspace_id', 'doc_id');

-- List all the lists in a space that belong to a folder
WITH folder_lists AS (
    SELECT folder_id, name FROM clickup_folders('space_id')
)
SELECT
    f.name AS folder_name,
    l.name AS list_name
FROM
    clickup_lists(f.folder_id) l,
    folder_lists f;
```

## Schema

### clickup_tasks

Returns a list of tasks for a given list. Takes a `list_id` as an argument.

| Column index | Column name    | type    |
| ------------ | -------------- | ------- |
| 0            | task_id        | TEXT    |
| 1            | description    | TEXT    |
| 2            | status         | TEXT    |
| 3            | order_index    | INTEGER |
| 4            | created_at     | TEXT    |
| 5            | updated_at     | TEXT    |
| 6            | closed_at      | TEXT    |
| 7            | done_at        | TEXT    |
| 8            | created_by     | TEXT    |
| 9            | started_at     | TEXT    |
| 10           | due_at         | TEXT    |
| 11           | estimated_time | INTEGER |
| 12           | time_spent     | INTEGER |
| 13           | assignees      | TEXT    |
| 14           | watchers       | TEXT    |
| 15           | tags           | TEXT    |
| 16           | custom_fields  | TEXT    |
| 17           | parent         | TEXT    |
| 18           | project_id     | TEXT    |
| 19           | folder_id      | TEXT    |
| 20           | space_id       | TEXT    |
| 21           | url            | TEXT    |

### clickup_docs

Returns a list of documents for a given workspace. Takes a `workspace_id` as an argument.

| Column index | Column name | type    |
| ------------ | ----------- | ------- |
| 0            | doc_id      | TEXT    |
| 1            | created_at  | TEXT    |
| 2            | updated_at  | TEXT    |
| 3            | name        | TEXT    |
| 4            | parent_id   | TEXT    |
| 5            | creator_id  | TEXT    |
| 6            | deleted     | INTEGER |

### clickup_pages

Returns a list of pages for a given document. Takes a `workspace_id` and a `doc_id` as arguments.

| Column index | Column name | type    |
| ------------ | ----------- | ------- |
| 0            | page_id     | TEXT    |
| 1            | created_at  | TEXT    |
| 2            | updated_at  | TEXT    |
| 3            | name        | TEXT    |
| 4            | creator_id  | TEXT    |
| 5            | content     | TEXT    |
| 6            | archived    | INTEGER |
| 7            | deleted     | INTEGER |
| 8            | protected   | INTEGER |

### clickup_folders

Returns a list of folders for a given space. Takes a `space_id` as an argument.

| Column index | Column name | type    |
| ------------ | ----------- | ------- |
| 0            | folder_id   | TEXT    |
| 1            | name        | TEXT    |
| 2            | archived    | INTEGER |

### clickup_lists

Returns a list of lists for a given folder. Takes a `folder_id` as an argument.

| Column index | Column name | type    |
| ------------ | ----------- | ------- |
| 0            | list_id     | TEXT    |
| 1            | name        | TEXT    |
| 2            | order_index | TEXT    |
| 3            | description | TEXT    |
| 4            | task_count  | INTEGER |
| 5            | due_at      | TEXT    |
| 6            | start_at    | TEXT    |
| 7            | archived    | INTEGER |

### clickup_whoami

Returns information about the authenticated user.

| Column index | Column name     | type    |
| ------------ | --------------- | ------- |
| 0            | id              | INTEGER |
| 1            | username        | TEXT    |
| 2            | email           | TEXT    |
| 3            | color           | TEXT    |
| 4            | profile_picture | TEXT    |
| 5            | initials        | TEXT    |
| 6            | week_start_day  | INTEGER |
| 7            | timezone        | TEXT    |

## Limitations

- The plugin caches data for 5 minutes. If you want to clear the cache, you can run `SELECT clear_plugin_cache('clickup');` and then restart anyquery.
- The plugin is read-only. You can't insert, update, or delete data.
