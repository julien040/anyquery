---
title: "How to list the oldest branches in a git repository?"
description: "Learn how to list the oldest branches in a Git repository using Anyquery. This tutorial covers prerequisites, installation, and SQL queries for both local and remote repos."
---

# How to List the Oldest Branches in a Git Repository

In this tutorial, we will demonstrate how to list the oldest branches in a Git repository using Anyquery. Anyquery is a versatile SQL query engine that can query various data sources, including Git repositories.

## Prerequisites

Before starting, ensure you have the following:

- [Anyquery installed](https://anyquery.dev/docs/#installation)
- The Git plugin installed (`anyquery install git`)
- A local or remote Git repository to query

## Step 1: Install the Git Plugin

First, install the Git plugin for Anyquery if you haven't already:

```bash
anyquery install git
```

## Step 2: Query the Branches

To list the oldest branches in a repository, we need to query the branches and sort them by their creation date. We will use the `git_branches` table provided by the Git plugin.

Here is the SQL query to list the oldest branches in a local Git repository:

```sql
SELECT full_name, name, MIN(author_date) as creation_date
FROM git_branches('path/to/local/git/repo')
JOIN git_commits('path/to/local/git/repo') ON git_branches.hash = git_commits.hash
GROUP BY full_name, name
ORDER BY creation_date ASC;
```

Replace `'path/to/local/git/repo'` with the path to your local Git repository.

If you want to query a remote Git repository, use the repository URL instead:

```sql
SELECT full_name, name, MIN(author_date) as creation_date
FROM git_branches('https://github.com/user/repo.git')
JOIN git_commits('https://github.com/user/repo.git') ON git_branches.hash = git_commits.hash
GROUP BY full_name, name
ORDER BY creation_date ASC;
```

Replace `'https://github.com/user/repo.git'` with the URL of your remote Git repository.

## Step 3: Run the Query

To run the query, use the `anyquery` command-line tool:

```bash
anyquery -q "SELECT full_name, name, MIN(author_date) as creation_date FROM git_branches('path/to/local/git/repo') JOIN git_commits('path/to/local/git/repo') ON git_branches.hash = git_commits.hash GROUP BY full_name, name ORDER BY creation_date ASC;"
```

Or for a remote Git repository:

```bash
anyquery -q "SELECT full_name, name, MIN(author_date) as creation_date FROM git_branches('https://github.com/user/repo.git') JOIN git_commits('https://github.com/user/repo.git') ON git_branches.hash = git_commits.hash GROUP BY full_name, name ORDER BY creation_date ASC;"
```

## Example Output

Here is an example of what the output might look like:

```plaintext
+-------------------------+----------------------+-------------------+
| full_name               | name                 | creation_date     |
+-------------------------+----------------------+-------------------+
| refs/heads/feature-old  | feature-old          | 2019-01-15 08:30  |
| refs/heads/feature-new  | feature-new          | 2019-03-20 14:45  |
| refs/heads/bugfix-old   | bugfix-old           | 2019-05-10 18:00  |
+-------------------------+----------------------+-------------------+
```

## Conclusion

You have successfully listed the oldest branches in a Git repository using Anyquery. By querying the `git_branches` table and joining it with the `git_commits` table, you can sort the branches by their creation date. This approach works for both local and remote Git repositories.

For more information on the Git plugin and its capabilities, refer to the [Git plugin documentation](https://anyquery.dev/integrations/git).
