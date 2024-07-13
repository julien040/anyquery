# GitHub plugin

A plugin to run SQL queries on GitHub data.

## Setup

To use this plugin, you need to install it first. You can do this by running the following command:

```bash
anyquery install github
```

Once done, anyquery will request you a GitHub Token. Head to [GitHub Personal Access Tokens](https://github.com/settings/tokens)
and create a new classic token with the scopes `repo`, `gist`, `read:packages`, and `read:org`. Copy the token and paste it in the terminal.

![Example to get a token](https://github.com/julien040/anyquery/blob/main/plugins/github/images/token-tutorial.png)

## Tables

### `github_my_repositories`

List the repositories that the authenticated user has access to.

```sql
SELECT * FROM github_my_repositories
```

#### Schema

| Column index | Column name                 | type    |
| ------------ | --------------------------- | ------- |
| 0            | id                          | INTEGER |
| 1            | node_id                     | TEXT    |
| 2            | owner                       | TEXT    |
| 3            | name                        | TEXT    |
| 4            | full_name                   | TEXT    |
| 5            | description                 | TEXT    |
| 6            | homepage                    | TEXT    |
| 7            | default_branch              | TEXT    |
| 8            | created_at                  | TEXT    |
| 9            | pushed_at                   | TEXT    |
| 10           | updated_at                  | TEXT    |
| 11           | html_url                    | TEXT    |
| 12           | clone_url                   | TEXT    |
| 13           | git_url                     | TEXT    |
| 14           | mirror_url                  | TEXT    |
| 15           | ssh_url                     | TEXT    |
| 16           | language                    | TEXT    |
| 17           | is_fork                     | INTEGER |
| 18           | forks_count                 | INTEGER |
| 19           | network_count               | INTEGER |
| 20           | open_issues_count           | INTEGER |
| 21           | stargazers_count            | INTEGER |
| 22           | subscribers_count           | INTEGER |
| 23           | size                        | INTEGER |
| 24           | allow_rebase_merge          | INTEGER |
| 25           | allow_update_branch         | INTEGER |
| 26           | allow_squash_merge          | INTEGER |
| 27           | allow_merge_commit          | INTEGER |
| 28           | allow_auto_merge            | INTEGER |
| 29           | allow_forking               | INTEGER |
| 30           | web_commit_signoff_required | INTEGER |
| 31           | delete_branch_on_merge      | INTEGER |
| 32           | topics                      | TEXT    |
| 33           | custom_properties           | TEXT    |
| 34           | archived                    | INTEGER |
| 35           | disabled                    | INTEGER |
| 36           | visibility                  | TEXT    |

### `github_repositories_from_user`

List the repositories from a specific user.

```sql
SELECT * FROM github_repositories_from_user('torvalds');
SELECT * FROM github_repositories_from_user WHERE user = 'torvalds';
```

#### Schema

| Column index | Column name                 | type    |
| ------------ | --------------------------- | ------- |
| 0            | id                          | INTEGER |
| 1            | node_id                     | TEXT    |
| 2            | owner                       | TEXT    |
| 3            | name                        | TEXT    |
| 4            | full_name                   | TEXT    |
| 5            | description                 | TEXT    |
| 6            | homepage                    | TEXT    |
| 7            | default_branch              | TEXT    |
| 8            | created_at                  | TEXT    |
| 9            | pushed_at                   | TEXT    |
| 10           | updated_at                  | TEXT    |
| 11           | html_url                    | TEXT    |
| 12           | clone_url                   | TEXT    |
| 13           | git_url                     | TEXT    |
| 14           | mirror_url                  | TEXT    |
| 15           | ssh_url                     | TEXT    |
| 16           | language                    | TEXT    |
| 17           | is_fork                     | INTEGER |
| 18           | forks_count                 | INTEGER |
| 19           | network_count               | INTEGER |
| 20           | open_issues_count           | INTEGER |
| 21           | stargazers_count            | INTEGER |
| 22           | subscribers_count           | INTEGER |
| 23           | size                        | INTEGER |
| 24           | allow_rebase_merge          | INTEGER |
| 25           | allow_update_branch         | INTEGER |
| 26           | allow_squash_merge          | INTEGER |
| 27           | allow_merge_commit          | INTEGER |
| 28           | allow_auto_merge            | INTEGER |
| 29           | allow_forking               | INTEGER |
| 30           | web_commit_signoff_required | INTEGER |
| 31           | delete_branch_on_merge      | INTEGER |
| 32           | topics                      | TEXT    |
| 33           | custom_properties           | TEXT    |
| 34           | archived                    | INTEGER |
| 35           | disabled                    | INTEGER |
| 36           | visibility                  | TEXT    |

### `github_commits_from_repository`

List the commits from a specific repository.

```sql
SELECT * FROM github_commits_from_repository('julien040/anyquery');
SELECT * FROM github_commits_from_repository WHERE repository = 'julien040/anyquery';
```

#### Schema

| Column index | Column name     | type |
| ------------ | --------------- | ---- |
| 0            | sha             | TEXT |
| 1            | committer       | TEXT |
| 2            | committer_email | TEXT |
| 3            | committer_date  | TEXT |
| 4            | author          | TEXT |
| 5            | author_email    | TEXT |
| 6            | author_date     | TEXT |
| 7            | message         | TEXT |
| 8            | html_url        | TEXT |

### `github_issues_from_repository`

List the issues from a specific repository.

```sql
SELECT * FROM github_issues_from_repository('julien040/gut');
SELECT * FROM github_issues_from_repository WHERE repository = 'julien040/gut';
```

#### Schema

| Column index | Column name  | type    |
| ------------ | ------------ | ------- |
| 0            | id           | INTEGER |
| 1            | number       | INTEGER |
| 2            | title        | TEXT    |
| 3            | body         | TEXT    |
| 4            | state        | TEXT    |
| 5            | state_reason | TEXT    |
| 6            | by           | TEXT    |
| 7            | assignees    | TEXT    |
| 8            | labels       | TEXT    |
| 9            | closed_at    | TEXT    |
| 10           | closed_by    | TEXT    |
| 11           | created_at   | TEXT    |
| 12           | updated_at   | TEXT    |
| 13           | url          | TEXT    |
| 14           | reactions    | TEXT    |
| 15           | draft        | INTEGER |
| 16           | locked       | INTEGER |

### `github_pull_requests_from_repository`

List the pull requests from a specific repository.

```sql
SELECT * FROM github_pull_requests_from_repository('sindresorhus/awesome');
SELECT * FROM github_pull_requests_from_repository WHERE repository = 'sindresorhus/awesome';
```

#### Schema

| Column index | Column name | type    |
| ------------ | ----------- | ------- |
| 0            | id          | INTEGER |
| 1            | number      | INTEGER |
| 2            | title       | TEXT    |
| 3            | body        | TEXT    |
| 4            | state       | TEXT    |
| 5            | by          | TEXT    |
| 6            | assignees   | TEXT    |
| 7            | labels      | TEXT    |
| 8            | closed_at   | TEXT    |
| 9            | created_at  | TEXT    |
| 10           | updated_at  | TEXT    |
| 11           | merged_at   | TEXT    |
| 12           | merged_by   | TEXT    |
| 13           | url         | TEXT    |

### `github_releases_from_repository`

List the releases from a specific repository.

```sql
SELECT * FROM github_releases_from_repository('julien040/gut');
SELECT * FROM github_releases_from_repository WHERE repository = 'julien040/gut';
```

#### Schema

| Column index | Column name  | type    |
| ------------ | ------------ | ------- |
| 0            | id           | INTEGER |
| 1            | name         | TEXT    |
| 2            | tag          | TEXT    |
| 3            | body         | TEXT    |
| 4            | created_at   | TEXT    |
| 5            | published_at | TEXT    |
| 6            | by           | TEXT    |
| 7            | url          | TEXT    |
| 8            | assets       | TEXT    |

### `github_branches_from_repository`

List the branches from a specific repository.

```sql
SELECT * FROM github_branches_from_repository('julien040/gut');
SELECT * FROM github_branches_from_repository WHERE repository = 'julien040/gut';
```

#### Schema

| Column index | Column name | type    |
| ------------ | ----------- | ------- |
| 0            | name        | TEXT    |
| 1            | commit_sha  | TEXT    |
| 2            | protected   | INTEGER |
| 3            | url         | TEXT    |

### `github_contributors_from_repository`

List the last 100 contributors and their stats from a specific repository.

```sql
SELECT * FROM github_contributors_from_repository('julien040/gut');
SELECT * FROM github_contributors_from_repository WHERE repository = 'julien040/gut';
```

#### Schema

| Column index | Column name     | type    |
| ------------ | --------------- | ------- |
| 0            | name            | TEXT    |
| 1            | contributor_url | TEXT    |
| 2            | additions       | INTEGER |
| 3            | deletions       | INTEGER |
| 4            | commits         | INTEGER |

### `github_tags_from_repository`

List the tags from a specific repository.

```sql
SELECT * FROM github_tags_from_repository('julien040/gut');
SELECT * FROM github_tags_from_repository WHERE repository = 'julien040/gut';
```

#### Schema

| Column index | Column name | type |
| ------------ | ----------- | ---- |
| 0            | name        | TEXT |
| 1            | commit_sha  | TEXT |
| 2            | commit_url  | TEXT |

### `github_followers_from_user`

List the followers of a specific user.

```sql
SELECT * FROM github_followers_from_user('nalgeon');
SELECT * FROM github_followers_from_user WHERE user = 'nalgeon';
```

#### Schema

| Column index | Column name  | type |
| ------------ | ------------ | ---- |
| 0            | follower     | TEXT |
| 1            | follower_url | TEXT |

### `github_following_from_user`

List the following of a specific user.

```sql
SELECT * FROM github_following_from_user('asg017');
SELECT * FROM github_following_from_user WHERE user = 'asg017';
```

#### Schema

| Column index | Column name  | type |
| ------------ | ------------ | ---- |
| 0            | follower     | TEXT |
| 1            | follower_url | TEXT |

### `github_my_followers`

List the followers of the authenticated user.

```sql
SELECT * FROM github_my_followers;
```

#### Schema

| Column index | Column name  | type |
| ------------ | ------------ | ---- |
| 0            | follower     | TEXT |
| 1            | follower_url | TEXT |

### `github_my_following`

List the following of the authenticated user.

```sql
SELECT * FROM github_my_following;
```

#### Schema

| Column index | Column name  | type |
| ------------ | ------------ | ---- |
| 0            | follower     | TEXT |
| 1            | follower_url | TEXT |

### `github_stars_from_user`

List the stars of a specific user.

```sql
SELECT * FROM github_stars_from_user('rauchg');
SELECT * FROM github_stars_from_user WHERE user = 'rauchg';
```

#### Schema

| Column index | Column name                 | type    |
| ------------ | --------------------------- | ------- |
| 0            | starred_at                  | TEXT    |
| 1            | id                          | INTEGER |
| 2            | node_id                     | TEXT    |
| 3            | owner                       | TEXT    |
| 4            | name                        | TEXT    |
| 5            | full_name                   | TEXT    |
| 6            | description                 | TEXT    |
| 7            | homepage                    | TEXT    |
| 8            | default_branch              | TEXT    |
| 9            | created_at                  | TEXT    |
| 10           | pushed_at                   | TEXT    |
| 11           | updated_at                  | TEXT    |
| 12           | html_url                    | TEXT    |
| 13           | clone_url                   | TEXT    |
| 14           | git_url                     | TEXT    |
| 15           | mirror_url                  | TEXT    |
| 16           | ssh_url                     | TEXT    |
| 17           | language                    | TEXT    |
| 18           | is_fork                     | INTEGER |
| 19           | forks_count                 | INTEGER |
| 20           | network_count               | INTEGER |
| 21           | open_issues_count           | INTEGER |
| 22           | stargazers_count            | INTEGER |
| 23           | subscribers_count           | INTEGER |
| 24           | size                        | INTEGER |
| 25           | allow_rebase_merge          | INTEGER |
| 26           | allow_update_branch         | INTEGER |
| 27           | allow_squash_merge          | INTEGER |
| 28           | allow_merge_commit          | INTEGER |
| 29           | allow_auto_merge            | INTEGER |
| 30           | allow_forking               | INTEGER |
| 31           | web_commit_signoff_required | INTEGER |
| 32           | delete_branch_on_merge      | INTEGER |
| 33           | topics                      | TEXT    |
| 34           | custom_properties           | TEXT    |
| 35           | archived                    | INTEGER |
| 36           | disabled                    | INTEGER |
| 37           | visibility                  | TEXT    |

### `github_my_stars`

List the stars of the authenticated user.

```sql
SELECT * FROM github_my_stars;
```

#### Schema

Identical to [github_stars_from_user](#github_stars_from_user).

### `github_gists_from_user`

List the public gists of a specific user.

```sql
SELECT * FROM github_gists_from_user('simonw');
SELECT * FROM github_gists_from_user WHERE user = 'simonw';
```

#### Schema

| Column index | Column name | type    |
| ------------ | ----------- | ------- |
| 0            | id          | TEXT    |
| 1            | gist_url    | TEXT    |
| 2            | by          | TEXT    |
| 3            | user_url    | TEXT    |
| 4            | description | TEXT    |
| 5            | comments    | INTEGER |
| 6            | public      | INTEGER |
| 7            | created_at  | TEXT    |
| 8            | updated_at  | TEXT    |

### `github_my_gists`

List the public and private gists of the authenticated user.

```sql
SELECT * FROM github_my_gists;
```

#### Schema

Identical to [github_gists_from_user](#github_gists_from_user).

### `github_comments_from_issue`

List the comments from a specific issue or pull request. If the issue is 0, it will list the comments of all issues and pull requests.

```sql
SELECT * FROM github_comments_from_issue('julien040/gut', 56);
SELECT * FROM github_comments_from_issue WHERE repository = 'julien040/gut' AND issue = 56;
```

#### Schema

| Column index | Column name        | type |
| ------------ | ------------------ | ---- |
| 0            | id                 | TEXT |
| 1            | body               | TEXT |
| 2            | by                 | TEXT |
| 3            | user_url           | TEXT |
| 4            | created_at         | TEXT |
| 5            | updated_at         | TEXT |
| 6            | author_association | TEXT |
| 7            | reactions          | TEXT |
| 8            | url                | TEXT |

## Caveats

- The plugin is limited to 5000 requests per hour. If you reach this limit, you will have to wait an hour before making new requests. This is a limitation from the GitHub API.
