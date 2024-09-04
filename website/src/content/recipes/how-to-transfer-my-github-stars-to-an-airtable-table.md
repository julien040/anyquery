---
title: "How to transfer my GitHub stars to an Airtable table?"
description: "Learn to transfer starred GitHub repositories to an Airtable table using Anyquery. This tutorial covers prerequisites, plugin configuration, and the SQL query needed for data transfer."
---

# How to Transfer Your GitHub Stars to an Airtable Table

In this tutorial, you'll learn how to transfer your starred GitHub repositories to an Airtable table using Anyquery. This is useful for organizing and analyzing your GitHub stars in a more structured manner.

## Prerequisites

Before starting, ensure you have the following:

- A working [installation of Anyquery](https://anyquery.dev/docs/#installation).
- A personal access token from GitHub with the `repo`, `user`, and `read:org` scopes. [GitHub integration guide](https://anyquery.dev/integrations/github).
- An Airtable API key and a table created to store your data. [Airtable integration guide](https://anyquery.dev/integrations/airtable). 

## Step 1: Install and Configure the Plugins

First, let's install the GitHub and Airtable plugins for Anyquery.

```bash
anyquery install github
anyquery install airtable
```

Next, you need to configure the plugins with your GitHub token and Airtable API key.

### GitHub Plugin Configuration

You'll be prompted to provide your GitHub token when installing the GitHub plugin. Create a token [here](https://github.com/settings/tokens) if you haven't already. Ensure it has `repo`, `user`, and `read:org` scopes.

### Airtable Plugin Configuration

You'll also be prompted to provide the Airtable API key, base ID, and table name. Here’s how to find them:
1. **Airtable API key**: Go to [Airtable API](https://airtable.com/account) and generate an API key.
2. **Base ID**: Find it in the URL of your Airtable base (e.g., `https://airtable.com/appXXXXXXXXXXXXXX` where `appXXXXXXXXXXXXXX` is your base ID).
3. **Table Name**: The name of the table you created in Airtable to store the data.

Ensure your Airtable table has the following columns:
- `Name` (Single line text)
- `Owner` (Single line text)
- `Stars` (Number)
- `Description` (Long text)
- `URL` (URL)

## Step 2: Transfer GitHub Stars to Airtable

Now, let's transfer your GitHub stars to Airtable with a SQL query.

First, ensure your connections are working:

```bash
anyquery -q "SELECT * FROM github_my_stars LIMIT 1; SELECT * FROM airtable_table LIMIT 1"
```

If both queries return results, you are ready to transfer the data.

### SQL Query to Transfer Data

```bash
anyquery -q "INSERT INTO airtable_table (Name, Owner, Stars, Description, URL) SELECT name, owner, stargazers_count, description, html_url FROM github_my_stars"
```

This query inserts your starred repositories into the Airtable table.

## Conclusion

You have successfully transferred your GitHub stars to an Airtable table using Anyquery. Now you can organize and analyze your starred repositories in Airtable. For more information on filtering, exporting, and data manipulation, refer to the [Anyquery documentation](https://anyquery.dev/docs/usage/).

Happy querying!
