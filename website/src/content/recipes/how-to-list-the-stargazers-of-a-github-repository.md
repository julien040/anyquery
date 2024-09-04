---
title: "How to list the stargazers of a GitHub repository?"
description: "Learn how to list GitHub repository stargazers using Anyquery's GitHub plugin. Follow steps to install, configure, and run SQL queries for detailed insights."
---

# How to List the Stargazers of a GitHub Repository

Anyquery is a SQL query engine that allows you to run SQL queries on a variety of data sources, including GitHub. In this tutorial, we'll show you how to list the stargazers of a GitHub repository using Anyquery.

## Introduction

Anyquery uses plugins to extend its functionalities. The GitHub plugin allows you to query GitHub data using SQL. Before we begin, ensure you have Anyquery installed and the GitHub plugin configured.

### What is Anyquery?

Anyquery is a SQL query engine that can query almost any data source, including databases, APIs, and files. For more information, visit the [official documentation](https://anyquery.dev/docs/#installation).

### Installing Anyquery

You can install Anyquery by following the instructions in the [installation guide](https://anyquery.dev/docs/#installation).

## Step 1: Install and Configure the GitHub Plugin

First, we need to install the GitHub plugin for Anyquery. This plugin allows you to interact with GitHub data using SQL.

```bash
anyquery install github
```

During the installation process, you'll be prompted to provide a GitHub personal access token. You can create one by following the instructions in the [GitHub integration guide](https://anyquery.dev/integrations/github).

Once the plugin is installed, you can configure it by providing the required token. This setup allows Anyquery to access GitHub data.

## Step 2: List the Stargazers of a Repository

To list the stargazers of a GitHub repository, we will use the `github_stargazers_from_repository` table provided by the GitHub plugin. This table requires the `owner/repo` argument, which specifies the repository you want to query.

### Example Query

Let's list the stargazers of the `octocat/Hello-World` repository:

```sql
SELECT * FROM github_stargazers_from_repository('octocat/Hello-World');
```

This query retrieves all the stargazers of the specified repository. You can run the query using Anyquery as follows:

```bash
anyquery -q "SELECT * FROM github_stargazers_from_repository('octocat/Hello-World');"
```

### Filter and Sort Results

You can also filter and sort the results to get more specific information. For example, to list the stargazers who starred the repository in the last month, you can use the following query:

```sql
SELECT * FROM github_stargazers_from_repository('octocat/Hello-World')
WHERE starred_at > datetime('now', '-1 month')
ORDER BY starred_at DESC;
```

Run the query using:

```bash
anyquery -q "SELECT * FROM github_stargazers_from_repository('octocat/Hello-World') WHERE starred_at > datetime('now', '-1 month') ORDER BY starred_at DESC;"
```

## Additional Information

### Using the MySQL Server Mode

If you prefer to use your favorite MySQL client, you can run Anyquery in MySQL server mode and connect to it remotely. Start the server as follows:

```bash
anyquery server
```

Then, connect to the server using a MySQL client and run the same queries.

### Exporting Results

You can export the results to various formats such as JSON, CSV, and HTML. For more information, refer to the [exporting results documentation](https://anyquery.dev/docs/usage/exporting-results).

## Conclusion

You have successfully listed the stargazers of a GitHub repository using Anyquery. You can now explore and query GitHub data using SQL. For more information on the GitHub plugin, visit the [plugin documentation](https://anyquery.dev/integrations/github).

---

Feel free to explore other features of Anyquery and the GitHub plugin to gain deeper insights into your GitHub data.
