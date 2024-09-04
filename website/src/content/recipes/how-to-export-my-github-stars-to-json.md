---
title: "How to export my GitHub stars to JSON?"
description: "Export your GitHub stars to JSON using Anyquery. Install the GitHub plugin, run the provided SQL query, and verify the export in the `github_stars.json` file."
---

# How to Export Your GitHub Stars to JSON

Anyquery is a SQL query engine that allows you to run SQL queries on pretty much anything, including exporting your GitHub stars to a JSON file. This tutorial will guide you through the process of exporting your GitHub stars to JSON using Anyquery.

## Prerequisites

Before you begin, ensure you have the following:

- A working [installation of Anyquery](https://anyquery.dev/docs/#installation)
- A GitHub personal access token with scopes: `repo`, `read:org`, `gist`, and `read:packages`. You can create one by following the [GitHub integration guide](https://anyquery.dev/integrations/github).

## Step 1: Install the GitHub Plugin

First, you need to install the GitHub plugin for Anyquery. Open your terminal and run:

```bash
anyquery install github
```

Follow the prompts to enter your GitHub personal access token.

## Step 2: Export GitHub Stars to JSON

Once the plugin is installed, you can run a SQL query to export your GitHub stars to JSON. Use the following command in your terminal:

```bash
anyquery -q "SELECT * FROM github_my_stars" --json > github_stars.json
```

This command does the following:
- `-q "SELECT * FROM github_my_stars"`: Runs a SQL query to select all columns from the `github_my_stars` table, which contains your starred repositories.
- `--json`: Specifies the output format as JSON.
- `> github_stars.json`: Redirects the output to a file named `github_stars.json`.

## Step 3: Verify the Export

To verify that your GitHub stars have been exported correctly, you can open the `github_stars.json` file using any text editor or JSON viewer. The file should contain a JSON array of objects, each representing a starred repository with various attributes.

## Conclusion

You have successfully exported your GitHub stars to a JSON file using Anyquery. You can further analyze or manipulate this data using any JSON-compatible tool or library.

For additional functionalities and advanced queries, refer to the [Anyquery documentation](https://anyquery.dev/docs/usage).

Happy querying!
