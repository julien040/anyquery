---
title: "How to see the GitHub stars obtained per day for a repository?"
description: "Learn to use Anyquery to track GitHub stars per day for a repository with simple SQL queries. This guide covers installation, setup, and example queries."
---

# How to See the GitHub Stars Obtained Per Day for a Repository

In this tutorial, we'll explore how to use Anyquery to see the GitHub stars obtained per day for a specific repository. Anyquery allows you to run SQL queries on various data sources, including GitHub.

## Prerequisites

Before starting, ensure you have the following:
- A working [installation of Anyquery](https://anyquery.dev/docs/#installation).
- The GitHub plugin installed and configured with your GitHub token. Follow the [GitHub plugin guide](https://anyquery.dev/integrations/github) to set it up.

## Step 1: Install the GitHub Plugin

First, install the GitHub plugin for Anyquery:

```bash
anyquery install github
```

You will be prompted to provide a GitHub token with the necessary scopes (`repo`, `read:org`, `gist`, `read:packages`). You can create a token by following the [GitHub integration guide](https://anyquery.dev/integrations/github).

## Step 2: Query for GitHub Stars Per Day

Once the GitHub plugin is installed and configured, you can query for the stars obtained per day for a specific repository. Here is the SQL query to do this:

```sql
SELECT 
    DATE(starred_at) AS day, 
    COUNT(*) AS stars 
FROM 
    github_stargazers_from_repository('owner/repo') 
GROUP BY 
    day 
ORDER BY 
    day;
```

Replace `'owner/repo'` with the actual owner and repository name you want to analyze. For example, to analyze the stars for the `vercel/next.js` repository, the query will be:

```sql
SELECT 
    DATE(starred_at) AS day, 
    COUNT(*) AS stars 
FROM 
    github_stargazers_from_repository('vercel/next.js') 
GROUP BY 
    day 
ORDER BY 
    day;
```

## Example of Running the Query

To run the query, open a terminal and execute the following command:

```bash
anyquery -q "SELECT DATE(starred_at) AS day, COUNT(*) AS stars FROM github_stargazers_from_repository('vercel/next.js') GROUP BY day ORDER BY day;"
```

This command will output the number of stars obtained per day for the `vercel/next.js` repository.

## Conclusion

You have successfully queried the GitHub stars obtained per day for a specific repository using Anyquery. Now you can explore and analyze the star trends for any GitHub repository. For more information on the GitHub plugin and other features, refer to the [GitHub plugin documentation](https://anyquery.dev/integrations/github).
