---
title: "How to update my brave tabs matching a specific URL?"
description: "Learn to update Brave browser tabs matching a specific URL using Anyquery. Follow steps to install the Brave plugin, query tabs, and update URLs with SQL commands."
---

# How to Update Brave Tabs Matching a Specific URL

This tutorial will guide you on how to update Brave browser tabs that match a specific URL using Anyquery. Anyquery is a powerful SQL query engine that allows you to run queries on various data sources, including browser tabs.

## Introduction to Anyquery

Anyquery enables you to execute SQL queries on diverse data sources such as databases, APIs, and files. You can install Anyquery by following the instructions on the [installation page](https://anyquery.dev/docs/#installation).

### Prerequisites

Before proceeding, ensure you have:

- A working installation of Anyquery
- Brave browser installed on your machine
- The Brave plugin installed in Anyquery (`anyquery install brave`)

## Step 1: Install the Brave Plugin

If you haven't already installed the Brave plugin, run the following command:

```bash
anyquery install brave
```

Refer to the [Brave plugin documentation](https://anyquery.dev/integrations/brave) if you need more information.

## Step 2: Launch Anyquery

Start Anyquery by running:

```bash
anyquery
```

## Step 3: Query Existing Brave Tabs

First, let's check the current open tabs in Brave:

```sql
SELECT * FROM brave_tabs;
```

This will list all the tabs currently open in your Brave browser.

## Step 4: Update Tabs Matching a Specific URL

To update tabs that match a specific URL, you can use the SQL `UPDATE` statement. For example, if you want to update all tabs with the URL `https://example.com` to `https://example.org`, run:

```sql
UPDATE brave_tabs 
SET url = 'https://example.org' 
WHERE url = 'https://example.com';
```

This command will change the URL of all open tabs currently pointing to `https://example.com` to `https://example.org`.

## Example: Changing URL of All GitHub Tabs

To change all GitHub URLs from `https://github.com` to `https://github.new`, you can run:

```sql
UPDATE brave_tabs 
SET url = 'https://github.new' 
WHERE url = 'https://github.com';
```

## Conclusion

You have successfully updated Brave tabs matching a specific URL using Anyquery. This versatile tool allows you to manage and manipulate your browser tabs with SQL. For more advanced queries and functionalities, refer to the [Anyquery documentation](https://anyquery.dev/docs/usage/*) and the [Brave plugin documentation](https://anyquery.dev/integrations/brave).

Happy querying!
