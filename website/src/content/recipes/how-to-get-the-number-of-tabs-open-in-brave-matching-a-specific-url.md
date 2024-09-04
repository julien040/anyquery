---
title: "How to get the number of tabs open in brave matching a specific URL?"
description: "Learn how to use Anyquery to find the number of open Brave browser tabs matching a specific URL with a simple SQL query. Perfect for effective tab management."
---

# How to Get the Number of Tabs Open in Brave Matching a Specific URL

Anyquery is a powerful SQL query engine that allows you to query data from various sources, including web browsers like Brave. This tutorial will guide you through the process of finding the number of open tabs in Brave that match a specific URL.

## Prerequisites

Before starting, ensure you have the following:
- A working installation of Anyquery. Follow the [installation guide](https://anyquery.dev/docs/#installation) if you haven't installed it yet.
- The Brave plugin installed. You can install it with the following command:

```bash
anyquery install brave
```

## Step 1: Launch Brave

Ensure your Brave browser is running and you have the tabs open that you want to query.

## Step 2: Run a Query to Get the Number of Tabs Matching a Specific URL

Open your terminal and run the following SQL query with Anyquery to find out the number of open tabs in Brave that match a specific URL:

```bash
anyquery -q "SELECT count(*) as tab_count FROM brave_tabs WHERE url LIKE '%specific-url%'"
```

Replace `%specific-url%` with the URL you are looking for. For example, if you want to find tabs matching `example.com`, the query would look like this:

```bash
anyquery -q "SELECT count(*) as tab_count FROM brave_tabs WHERE url LIKE '%example.com%'"
```

This query will return the number of tabs that contain `example.com` in their URL.

## Example Query

Here is an example of querying to find the number of tabs open in Brave that match `github.com`:

```bash
anyquery -q "SELECT count(*) as tab_count FROM brave_tabs WHERE url LIKE '%github.com%'"
```

This will output the count of Brave tabs that have `github.com` in their URL.

## Conclusion

You have successfully queried the number of open tabs in Brave matching a specific URL using Anyquery. You can now use this technique to monitor and manage your browser tabs more effectively. For more information about Anyquery and its features, refer to the [official documentation](https://anyquery.dev/docs/usage/running-queries).

Feel free to explore other functionalities of the Brave plugin by running different SQL queries to manage your browser tabs.
