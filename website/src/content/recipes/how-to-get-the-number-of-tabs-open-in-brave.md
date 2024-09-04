---
title: "How to get the number of tabs open in brave?"
description: "Learn how to use Anyquery to get the number of tabs open in Brave with simple SQL commands. Follow steps to install the Brave plugin, connect, and run queries easily."
---

# How to Get the Number of Tabs Open in Brave

Anyquery is a SQL query engine that allows you to run SQL queries on virtually anything, including open tabs in browsers like Brave. In this tutorial, we will learn how to get the number of tabs open in Brave using Anyquery.

## Prerequisites

Before starting, ensure you have the following:

- A working [installation of Anyquery](https://anyquery.dev/docs/#installation)
- The Brave plugin installed

## Step 1: Install the Brave Plugin

First, if you haven't already, you need to install the Brave plugin:

```bash
anyquery install brave
```

## Step 2: Connect to Brave

Anyquery will connect directly to the Brave browser to fetch the tabs information. Ensure that Brave is running on your machine.

## Step 3: Query the Open Tabs

To get the number of tabs open in Brave, you can run a simple SQL query. Open your terminal and execute the following command:

```bash
anyquery -q "SELECT COUNT(*) as open_tabs FROM brave_tabs"
```

This query will return the number of open tabs in Brave.

## Example Output

Here's an example of what the output might look like:

```bash
open_tabs
----------
         8
```

This output indicates that there are 8 tabs open in Brave.

## Additional Information

You can also get more detailed information about the open tabs using different queries. For example, to list all the tabs' titles and URLs:

```bash
anyquery -q "SELECT title, url FROM brave_tabs"
```

See the [Brave plugin documentation](https://anyquery.dev/integrations/brave) for more information on the available columns and other functionalities.

## Conclusion

You have successfully queried the number of tabs open in Brave using Anyquery. This powerful SQL query engine allows you to interact with your browser tabs and perform various operations. Explore more functionalities and integrate them into your workflows as needed.
