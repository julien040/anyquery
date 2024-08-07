# Todoist plugin

A plugin to explore your tasks on Todoist with SQL.

## Setup

```bash
anyquery install todoist
```

The plugin will ask you to pass a token to authenticate with Todoist. You can create a token at [https://app.todoist.com/app/settings/integrations/developer](https://app.todoist.com/app/settings/integrations/developer). Select the `developer` tab and copy the the API key. Paste it in the anyquery form.

## Tables

### todoist_active_tasks

List all your active tasks. The table supports insert/delete and select operations. When deleting a task, it'll close the task on Todoist.

#### Schema

| Column index | Column name   | type    |
| ------------ | ------------- | ------- |
| 0            | id            | TEXT    |
| 1            | assigner_id   | TEXT    |
| 2            | assignee_id   | TEXT    |
| 3            | project_id    | TEXT    |
| 4            | section_id    | TEXT    |
| 5            | parent_id     | TEXT    |
| 6            | order         | INTEGER |
| 7            | content       | TEXT    |
| 8            | description   | TEXT    |
| 9            | is_completed  | INTEGER |
| 10           | labels        | TEXT    |
| 11           | priority      | INTEGER |
| 12           | comment_count | INTEGER |
| 13           | creator_id    | TEXT    |
| 14           | created_at    | TEXT    |
| 15           | due           | TEXT    |
| 16           | url           | TEXT    |

## Caveats

- The plugin does not support updating tasks.
- The plugin does not do any caching and fetches the tasks every time you run a query. To avoid rate limiting, you can insert your tasks into a table and query that table instead. For example, `INSERT INTO my_tasks SELECT * FROM todoist_active_tasks` and then `SELECT * FROM my_tasks`.
- The plugin is subject to the rate limits of the Todoist API (1000 requests per 15 minutes). If you hit the rate limit, you can wait for 15 minutes or use the cache method mentioned above. Therefore, you cannot insert/delete tasks more than 1000 times in 15 minutes.
- Due date resolution is only per day and not per hour. I don't have a premium account to test the exact behavior of the API. If you have a premium account and can help me test, please reach out to me.
