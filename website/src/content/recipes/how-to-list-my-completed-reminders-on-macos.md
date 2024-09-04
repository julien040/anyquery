---
title: "How to list my completed reminders on macOS?"
description: "Learn how to list completed reminders on macOS using Anyquery. Install the Reminders plugin, run an SQL query, and export results in CSV or JSON formats."
---

# How to List Completed Reminders on macOS

Anyquery is a powerful SQL query engine that allows you to query various data sources, including Apple's Reminders app on macOS. In this tutorial, we will walk you through the steps to list your completed reminders using Anyquery.

## Prerequisites

Before you start, ensure you have the following:

- A working [installation of Anyquery](https://anyquery.dev/docs/#installation)
- The Reminders plugin installed (`anyquery install reminders`)

## Step 1: Install the Reminders Plugin

First, you need to install the Reminders plugin. Open your terminal and run the following command:

```bash
anyquery install reminders
```

When you run your first query, a popup will appear asking for permission to allow your terminal to control Reminders. Make sure to grant this permission for the plugin to work properly.

## Step 2: List Completed Reminders

To list your completed reminders, you can use a straightforward SQL query. Open your terminal and run:

```bash
anyquery -q "SELECT * FROM reminders_items WHERE completed = 1"
```

This query selects all reminders from the `reminders_items` table where the `completed` column is set to `1` (true).

## Step 3: Export Results

Optionally, you might want to export the results to a file, such as CSV or JSON. Here’s how you can do it:

### Export to CSV

```bash
anyquery -q "SELECT * FROM reminders_items WHERE completed = 1" --csv > completed_reminders.csv
```

### Export to JSON

```bash
anyquery -q "SELECT * FROM reminders_items WHERE completed = 1" --json > completed_reminders.json
```

## Conclusion

You have successfully queried and listed your completed reminders on macOS using Anyquery. You can now manipulate, export, and utilize this data as needed.

## Additional Information

Here is a brief overview of the `reminders_items` table schema:

| Column index | Column name | type    |
| ------------ | ----------- | ------- |
| 0            | id          | TEXT    |
| 1            | list        | TEXT    |
| 2            | name        | TEXT    |
| 3            | body        | TEXT    |
| 4            | completed   | INTEGER |
| 5            | due_date    | TEXT    |
| 6            | priority    | INTEGER |

For more information, visit the [official Anyquery documentation](https://anyquery.dev/docs/usage/troubleshooting/).

Happy querying!
