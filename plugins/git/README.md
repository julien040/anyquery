# Git plugin

Run SQL queries on a Git repository.

## Usage

```sql
-- Get all commits from a local repository
SELECT * FROM git_commits('path/to/repo');
-- Get the count of commits of gitster(Junio C Hamano) in the git repository
SELECT count(*) FROM git_commits('https://github.com/git/git.git') WHERE author_name='Junio C Hamano';
```

> ⚠️ To speed up the queries, you can run `git gc` on the repository. It can reduce query time by up to 10x.

## Installation

[You need to have `git` installed](https://git-scm.com/book/en/v2/Getting-Started-Installing-Git) and available in the PATH for remote repositories.

```bash
anyquery install git
```

## Tables

### `git_commits`

Returns the list of commits in the repository.

#### Schema

| Column index | Column name     | type |
| ------------ | --------------- | ---- |
| 0            | hash            | TEXT |
| 1            | author_name     | TEXT |
| 2            | author_email    | TEXT |
| 3            | author_date     | TEXT |
| 4            | committer_name  | TEXT |
| 5            | committer_email | TEXT |
| 6            | committer_date  | TEXT |
| 7            | message         | TEXT |

### `git_commits_diff`

Returns a row for each file changed in each commit. This function is slow (multiple minutes) for large repositories (e.g. [git/git](https://github.com/git/git.git)).

#### Example

```sql
-- Get the most modified files in the git repository
SELECT sum(addition)+sum(deletion) as "changes", file_name FROM git_commits_diff('/path/to/repo') GROUP BY file_name ORDER BY
"changes" DESC LIMIT 10;
-- Get the number of lines added per user in the git repository
SELECT sum(addition) as "addition", author_name FROM git_commits_diff('/path/to/repo') GROUP BY author_name ORDER BY "addition" DESC;
```

#### Schema

| Column index | Column name     | type    |
| ------------ | --------------- | ------- |
| 0            | hash            | TEXT    |
| 1            | author_name     | TEXT    |
| 2            | author_email    | TEXT    |
| 3            | author_date     | TEXT    |
| 4            | committer_name  | TEXT    |
| 5            | committer_email | TEXT    |
| 6            | committer_date  | TEXT    |
| 7            | message         | TEXT    |
| 8            | file_name       | TEXT    |
| 9            | addition        | INTEGER |
| 10           | deletion        | INTEGER |

### `git_branches`

Returns the list of branches in the repository.

#### Schema

| Column index | Column name | type |
| ------------ | ----------- | ---- |
| 0            | full_name   | TEXT |
| 1            | name        | TEXT |
| 2            | hash        | TEXT |

### `git_tags`

Returns the list of tags in the repository.

#### Schema

| Column index | Column name | type |
| ------------ | ----------- | ---- |
| 0            | full_name   | TEXT |
| 1            | name        | TEXT |
| 2            | hash        | TEXT |

### `git_remotes`

Returns the list of remotes in the repository.

#### Schema

| Column index | Column name | type    |
| ------------ | ----------- | ------- |
| 0            | name        | TEXT    |
| 1            | url         | TEXT    |
| 2            | is_mirror   | INTEGER |

### `git_references`

Returns the list of references (branches + tags + remotes) in the repository.

#### Schema

| Column index | Column name | type |
| ------------ | ----------- | ---- |
| 0            | full_name   | TEXT |
| 1            | name        | TEXT |
| 2            | hash        | TEXT |

## Caveats

- The plugin is not optimized for large repositories. It does not yet use caching or other optimizations.
- I haven't tested support on submodules.
