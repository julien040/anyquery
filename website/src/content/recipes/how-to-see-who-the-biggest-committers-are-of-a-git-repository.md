---
title: "How to see who the biggest committers are of a Git repository?"
description: "Analyze Git commit data with Anyquery. Learn to set up connections, run SQL queries to identify top committers, and visualize data using tools like Metabase."
---

# How to See Who the Biggest Committers are of a Git Repository

Anyquery is a SQL query engine that allows you to run SQL queries on pretty much anything. In this tutorial, we will use Anyquery to analyze a Git repository and see who the biggest committers are.

## Prerequisites

Before starting, ensure you have the following:

- A working [installation of Anyquery](https://anyquery.dev/docs/#installation)
- The Git plugin installed: `anyquery install git`
- A Git repository URL or a local path to a Git repository

## Step 1: Set up the connection

You can analyze a local Git repository or a remote one. If you are working with a local repository, ensure it is accessible. For a remote repository, ensure you have `git` installed and available in your `PATH`.

### Local Repository

If you have a local Git repository, you can use the following SQL query:

```sql
SELECT * FROM git_commits('/path/to/your/local/repo');
```

Replace `/path/to/your/local/repo` with the actual path to your Git repository.

### Remote Repository

If you are working with a remote repository, use the repository URL:

```sql
SELECT * FROM git_commits('https://github.com/user/repo.git');
```

Replace `https://github.com/user/repo.git` with the actual URL of the Git repository.

## Step 2: Analyze the Commits

To see who the biggest committers are, we need to aggregate the commits by author and count them. Here is the SQL query to do that:

```sql
SELECT author_name, COUNT(*) AS commit_count
FROM git_commits('/path/to/your/local/repo')
GROUP BY author_name
ORDER BY commit_count DESC
LIMIT 10;
```

Replace `/path/to/your/local/repo` with the path to your local repository or use the remote repository URL in the `git_commits` function if you are analyzing a remote repository.

This query will return the top 10 committers by the number of commits.

### Example

For example, to analyze the commits of the Git repository itself:

```sql
SELECT author_name, COUNT(*) AS commit_count
FROM git_commits('https://github.com/git/git.git')
GROUP BY author_name
ORDER BY commit_count DESC
LIMIT 10;
```

## Step 3: Visualize the Data

While Anyquery can provide the data, visualizing it can help you better understand the contributions. You can export the results to a CSV file and use your favorite data visualization tool.

### Exporting to CSV

Here is how you can export the results to a CSV file:

```bash
anyquery -q "SELECT author_name, COUNT(*) AS commit_count FROM git_commits('https://github.com/git/git.git') GROUP BY author_name ORDER BY commit_count DESC LIMIT 10" --csv > committers.csv
```

### Using Metabase

To visualize the data using Metabase, follow the [Metabase connection guide](https://anyquery.dev/integrations/metabase).

1. Add the database connection in Metabase using the Anyquery server URL.
2. Create a new question using the native query:
   ```sql
   SELECT author_name, COUNT(*) AS commit_count
   FROM git_commits('https://github.com/git/git.git')
   GROUP BY author_name
   ORDER BY commit_count DESC
   LIMIT 10;
   ```
3. Visualize the data using Metabase's visualization options.

## Conclusion

You have successfully analyzed a Git repository to see who the biggest committers are using Anyquery. You can now use this method to analyze any Git repository, whether it is local or remote. For more information about the Git plugin and its usage, refer to the [Git plugin documentation](https://anyquery.dev/docs/usage/querying-log/).
