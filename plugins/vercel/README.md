# Vercel

A plugin to explore your projects and deployments on Vercel.

## Setup

```bash
anyquery install vercel
```

Once done, anyquery will ask you to authenticate with Vercel. Create a token at [https://vercel.com/account/tokens](https://vercel.com/account/tokens) and paste it in the terminal.

If you have multiple accounts, you can create multiple profiles with `anyquery profile new default vercel`.

Also, each table can be filtered by teamID by adding a condtion in the WHERE clause. For example, `SELECT * FROM vercel_deployments WHERE team_id = 'team_123'` or by passing it as the last table argument `SELECT * FROM vercel_deployments(...otherArgs, 'team_123')`.

## Tables

### vercel_projects

| Column index | Column name       | type |
| ------------ | ----------------- | ---- |
| 0            | account_id        | TEXT |
| 1            | created_at        | TEXT |
| 2            | updated_at        | TEXT |
| 3            | framework         | TEXT |
| 4            | project_id        | TEXT |
| 5            | name              | TEXT |
| 6            | node_version      | TEXT |
| 7            | serverless_region | TEXT |

### vercel_deployments

You can filter by project_id or team_id. For example, `SELECT * FROM vercel_deployments('project_123')` or `SELECT * FROM vercel_deployments('project_123', 'team_123')`. These arguments are also available as columns `project_id` and `team_id` and can be used in the WHERE clause.

| Column index | Column name           | type |
| ------------ | --------------------- | ---- |
| 0            | id                    | TEXT |
| 1            | name                  | TEXT |
| 2            | url                   | TEXT |
| 3            | created_at            | TEXT |
| 4            | ready_at              | TEXT |
| 5            | building_at           | TEXT |
| 6            | source                | TEXT |
| 7            | state                 | TEXT |
| 8            | substate              | TEXT |
| 9            | type                  | TEXT |
| 10           | target                | TEXT |
| 11           | creator_email         | TEXT |
| 12           | creator_name          | TEXT |
| 13           | inspector_url         | TEXT |
| 14           | github_commit_sha     | TEXT |
| 15           | github_commit_author  | TEXT |
| 16           | github_commit_message | TEXT |

## Caveats

- The plugin is read-only and does not support creating or updating projects or deployments.
- The plugin caches the deployments for 2 minutes and the projects for an hour. If you want to refresh the cache, you can run `SELECT clear_plugin_cache('vercel')` and restart anyquery.
- The plugin cannot yet list domains, aliases, or secrets.
