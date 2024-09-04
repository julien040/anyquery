---
title: "How to list the newest branches in a git repository?"
description: "Learn to list the newest branches in a git repository using Anyquery. Follow steps to install the git plugin, run SQL queries, and export results to CSV."
---

# How to List the Newest Branches in a Git Repository

Anyquery is a SQL query engine that allows you to run SQL queries on virtually any data source, including git repositories. In this tutorial, we will show you how to list the newest branches in a git repository using Anyquery.

## Prerequisites

Before starting, ensure you have the following:

- A working installation of Anyquery. Refer to the [installation guide](https://anyquery.dev/docs/#installation) for more information.
- The `git` plugin installed in Anyquery.

To install the `git` plugin, run the following command:

```bash
anyquery install git
```

## Listing the Newest Branches

To list the newest branches in a git repository, use the `git_branches` table function. The `git_branches` function takes the path to the repository as an argument. For remote repositories, ensure that you have `git` installed and available in the PATH.

### Example Query

Let's start by listing all the branches in a local git repository and ordering them by their creation date. The creation date is inferred from the commit date of the latest commit in the branch.

```sql
SELECT name, hash, committer_date
FROM (
    SELECT name, hash,
           (SELECT committer_date FROM git_commits(repository_path) WHERE git_commits.hash = git_branches.hash) as committer_date
    FROM git_branches('path/to/repo')
)
ORDER BY committer_date DESC;
```

In this query:
- We first select the branch name and hash from the `git_branches` table.
- We use a subquery to get the `committer_date` from the latest commit in each branch using the `git_commits` table.
- Finally, we order the result by the `committer_date` in descending order to list the newest branches.

### Using Remote Repositories

If you want to list the newest branches from a remote repository, you can specify the URL of the repository. Ensure `git` is installed and available in the PATH.

```sql
SELECT name, hash, committer_date
FROM (
    SELECT name, hash,
           (SELECT committer_date FROM git_commits('https://github.com/user/repo.git') WHERE git_commits.hash = git_branches.hash) as committer_date
    FROM git_branches('https://github.com/user/repo.git')
)
ORDER BY committer_date DESC;
```

### Exporting Results

You can also export the results to different formats like CSV, JSON, and HTML. Here is an example to export the results to a CSV file:

```bash
anyquery -q "SELECT name, hash, committer_date FROM (SELECT name, hash, (SELECT committer_date FROM git_commits('https://github.com/user/repo.git') WHERE git_commits.hash = git_branches.hash) as committer_date FROM git_branches('https://github.com/user/repo.git')) ORDER BY committer_date DESC" --csv > branches.csv
```

## Conclusion

You have successfully listed the newest branches in a git repository using Anyquery. Now you can explore and analyze your git repositories using SQL. For more information on Anyquery and its features, refer to the [official documentation](https://anyquery.dev/docs/usage/troubleshooting/).
