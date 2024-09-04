---
title: "How to close my brave tabs matching a specific URL?"
description: "Learn how to close Brave Browser tabs matching a specific URL using Anyquery's SQL query engine. Follow simple steps to install the plugin and execute queries."
---

# How to Close Brave Tabs Matching a Specific URL

## Introduction

If you're using Brave Browser and want to close specific tabs matching a particular URL, Anyquery has got you covered. Anyquery is a SQL query engine that allows you to run SQL queries on almost anything, including browser tabs.

## Prerequisites

Before we dive in, ensure you have the following:
- A working installation of Anyquery. Follow the [installation guide](https://anyquery.dev/docs/#installation).
- Brave Browser installed on your machine.
- The Brave plugin installed in Anyquery.

## Step 1: Install the Brave Plugin

First, you need to install the Brave plugin for Anyquery. Open your terminal and run:

```bash
anyquery install brave
```

## Step 2: Allow Terminal to Control Brave

On the first usage, you'll be prompted to allow your terminal to control Brave Browser. This is required for the plugin to work correctly. Make sure you accept this permission.

## Step 3: Query and Close Specific Tabs

Once the Brave plugin is installed and configured, you can close tabs matching a specific URL using SQL queries.

### Example: Close Tabs Matching a Specific URL

To close tabs matching a specific URL, use the `DELETE` statement. For example, if you want to close all tabs with the URL 'https://example.com', run the following query:

```sql
DELETE FROM brave_tabs WHERE url = 'https://example.com';
```

You can run this query using Anyquery as follows:

```bash
anyquery -q "DELETE FROM brave_tabs WHERE url = 'https://example.com';"
```

## Summary

You have successfully installed the Brave plugin and closed specific tabs matching a URL using Anyquery. The steps are straightforward:
1. Install the Brave plugin.
2. Allow terminal to control Brave.
3. Use SQL queries to manage your tabs.

For more details and advanced usage, refer to the [Brave plugin documentation](https://anyquery.dev/integrations/brave).
