# Asana plugin

Query and manage your Asana tasks, projects, goals, and workspaces with SQL.

## Setup

```bash
anyquery install asana
```

The plugin will ask for your Asana Personal Access Token (PAT). To create one, go to [https://app.asana.com/0/my-apps](https://app.asana.com/0/my-apps) and click **Create new token**.

## Usage

Most tables require a parameter to scope the query. You can pass parameters either as a table function argument or via a `WHERE` clause:

```sql
-- Using table function syntax
SELECT * FROM asana_tasks('project_gid');

-- Using WHERE clause
SELECT * FROM asana_tasks WHERE project_id = 'project_gid';
```

To find GIDs, open the item in Asana and look at the URL. For example, in `https://app.asana.com/0/1234567890/list`, the workspace/project GID is `1234567890`.

You can also query `asana_workspaces` to discover your workspace GIDs, then `asana_projects` to find project GIDs.

```sql
-- Step 1: find your workspace GID
SELECT gid, name FROM asana_workspaces;

-- Step 2: find projects in that workspace
SELECT gid, name FROM asana_projects WHERE workspace_gid = '<workspace_gid>';

-- Step 3: query tasks in a project
SELECT * FROM asana_tasks WHERE project_id = '<project_gid>';
```

## Tables

### `asana_tasks`

List, create, update, and delete tasks within a project.

**Parameter:** `project_id` — the GID of the project (required).

#### Schema

| Column index | Column name   | Type    | Description                                              |
| ------------ | ------------- | ------- | -------------------------------------------------------- |
| 0            | project_id    | TEXT    | Parameter: GID of the project to query tasks from       |
| 1            | gid           | TEXT    | Globally unique task identifier (primary key)            |
| 2            | name          | TEXT    | The name of the task                                     |
| 3            | completed     | INTEGER | 1 if the task is completed, 0 otherwise                  |
| 4            | assignee      | TEXT    | GID of the assigned user                                 |
| 5            | due_at        | TEXT    | Due date (YYYY-MM-DD or YYYY-MM-DDTHH:MM:SS)             |
| 6            | notes         | TEXT    | Task description or notes                                |
| 7            | created_at    | TEXT    | Creation timestamp (RFC3339)                             |
| 8            | modified_at   | TEXT    | Last modification timestamp (RFC3339)                    |
| 9            | liked         | INTEGER | 1 if liked by the current user, 0 otherwise              |
| 10           | start_at      | TEXT    | Start date (YYYY-MM-DD)                                  |
| 11           | parent        | TEXT    | Parent task GID if this is a subtask                     |
| 12           | project       | TEXT    | Name of the project the task belongs to                  |
| 13           | section       | TEXT    | Section name (e.g. To do, In progress, Done)             |
| 14           | task_type     | TEXT    | Type of the task: task, milestone, or approval           |
| 15           | custom_fields | TEXT    | JSON object of custom fields                             |

#### Examples

```sql
-- List all incomplete tasks in a project
SELECT gid, name, due_at, assignee
FROM asana_tasks('1234567890')
WHERE completed = 0;

-- Mark a task as complete
UPDATE asana_tasks SET completed = 1 WHERE gid = '<task_gid>' AND project_id = '<project_gid>';

-- Create a new task
INSERT INTO asana_tasks (project_id, name, notes, due_at)
VALUES ('<project_gid>', 'Fix login bug', 'Users cannot log in on mobile', '2024-12-31');

-- Delete a task
DELETE FROM asana_tasks WHERE gid = '<task_gid>' AND project_id = '<project_gid>';

-- Count tasks by section
SELECT section, COUNT(*) AS task_count
FROM asana_tasks('<project_gid>')
GROUP BY section;

-- Read a custom field
SELECT name, custom_fields ->> '$.Priority' AS priority
FROM asana_tasks('<project_gid>');
```

---

### `asana_projects`

List all projects within a workspace.

**Parameter:** `workspace_gid` — the GID of the workspace (required).

#### Schema

| Column index | Column name   | Type    | Description                                              |
| ------------ | ------------- | ------- | -------------------------------------------------------- |
| 0            | workspace_gid | TEXT    | Parameter: GID of the workspace                         |
| 1            | gid           | TEXT    | Globally unique project identifier (primary key)         |
| 2            | name          | TEXT    | The name of the project                                  |
| 3            | owner         | TEXT    | Name of the project owner                                |
| 4            | created_at    | TEXT    | Creation timestamp (RFC3339)                             |
| 5            | modified_at   | TEXT    | Last modification timestamp (RFC3339)                    |
| 6            | archived      | INTEGER | 1 if the project is archived, 0 otherwise                |
| 7            | color         | TEXT    | The color assigned to the project                        |
| 8            | notes         | TEXT    | Free-form notes associated with the project              |
| 9            | workspace     | TEXT    | Name of the workspace the project belongs to             |
| 10           | team          | TEXT    | Name of the team the project belongs to                  |

#### Examples

```sql
-- List all active projects in a workspace
SELECT gid, name, owner, team
FROM asana_projects('<workspace_gid>')
WHERE archived = 0;

-- Find projects modified in the last 30 days
SELECT name, modified_at
FROM asana_projects('<workspace_gid>')
WHERE modified_at >= date('now', '-30 days')
ORDER BY modified_at DESC;
```

---

### `asana_goals`

List goals within a workspace. Requires an Asana Premium account or higher.

**Parameter:** `workspace_gid` — the GID of the workspace (optional).

#### Schema

| Column index | Column name   | Type    | Description                                              |
| ------------ | ------------- | ------- | -------------------------------------------------------- |
| 0            | workspace_gid | TEXT    | Parameter: filter goals by workspace GID                 |
| 1            | gid           | TEXT    | Globally unique goal identifier (primary key)            |
| 2            | name          | TEXT    | The name of the goal                                     |
| 3            | owner         | TEXT    | Name of the goal owner                                   |
| 4            | created_at    | TEXT    | Creation timestamp (RFC3339)                             |
| 5            | due_on        | TEXT    | Due date (YYYY-MM-DD)                                    |
| 6            | status        | TEXT    | Current status (e.g. on_track, at_risk, missed)          |
| 7            | notes         | TEXT    | Free-form notes associated with the goal                 |
| 8            | workspace     | TEXT    | Name of the workspace the goal belongs to                |

#### Examples

```sql
-- List all goals in a workspace
SELECT name, status, due_on
FROM asana_goals('<workspace_gid>');

-- Find at-risk or missed goals
SELECT name, owner, due_on
FROM asana_goals('<workspace_gid>')
WHERE status IN ('at_risk', 'missed');
```

---

### `asana_workspaces`

List all workspaces and organizations accessible to the authenticated user. No parameters required.

#### Schema

| Column index | Column name     | Type    | Description                                              |
| ------------ | --------------- | ------- | -------------------------------------------------------- |
| 0            | gid             | TEXT    | Globally unique workspace identifier (primary key)       |
| 1            | name            | TEXT    | The name of the workspace                                |
| 2            | is_organization | INTEGER | 1 if the workspace is an organization, 0 otherwise       |

#### Examples

```sql
-- List all workspaces
SELECT * FROM asana_workspaces;

-- Find organizations only
SELECT gid, name FROM asana_workspaces WHERE is_organization = 1;
```

## Caveats

- The plugin does not cache results. Every query fetches live data from the Asana API. To avoid rate limits on large datasets, consider inserting results into a local table: `INSERT INTO my_tasks SELECT * FROM asana_tasks('<project_gid>')`.
- The `asana_goals` table requires an Asana Premium account or higher. It will return an empty result set on free plans.
- Tasks are scoped to a project via `project_id`. To query tasks across all projects, run separate queries per project GID.
- Asana rate limits the API to 1,500 requests per minute. Queries that paginate through many results may be throttled.
- `INSERT` supports the `name` (required), `notes`, `due_at`, `start_at`, and `completed` columns. Other columns are ignored on insert.
- `UPDATE` supports `name`, `completed`, `due_at`, `notes`, and `start_at`. Other columns are ignored on update.
