---
title: "How to create brave tabs using a JSON file?"
description: "Learn how to automate creating new tabs in Brave using Anyquery by reading a JSON file. Follow the steps to install the plugin, create a JSON file, and execute SQL queries."
---

# How to Create Brave Tabs Using a JSON File with Anyquery

Anyquery is a versatile SQL query engine that allows you to run SQL queries on various data sources, including creating and managing browser tabs. In this tutorial, we will learn how to use Anyquery to create new tabs in the Brave browser by reading a JSON file.

## Prerequisites

Before starting, ensure you have the following:
- A working [installation of Anyquery](https://anyquery.dev/docs/#installation)
- Brave browser installed on your machine
- The Brave plugin installed

### Install the Brave Plugin

To install the Brave plugin, run the following command:

```bash
anyquery install brave
```

## Step 1: Create Your JSON File

Firstly, you need a JSON file containing the URLs of the tabs you want to create. Create a file named `tabs.json` with the following structure:

```json
[
  {
    "url": "https://example.com"
  },
  {
    "url": "https://docs.anyquery.dev"
  },
  {
    "url": "https://github.com"
  }
]
```

## Step 2: Read the JSON File with Anyquery

Anyquery can read JSON files and use the data to perform operations. To read the JSON file, use the `read_json` function.

```sql
SELECT * FROM read_json('tabs.json');
```

## Step 3: Create Brave Tabs from the JSON File

To create new tabs in Brave using the URLs from the JSON file, you will use the `INSERT INTO` statement to insert the data from the JSON file into the `brave_tabs` table.

Here is the SQL query to achieve that:

```bash
anyquery -q "INSERT INTO brave_tabs (url) SELECT url FROM read_json('tabs.json')"
```

## Step 4: Verify the Tabs

After running the above query, open your Brave browser to verify that the new tabs have been created with the URLs specified in the JSON file.

## Conclusion

You have successfully created new tabs in the Brave browser using a JSON file with Anyquery. This method can be particularly useful for automating the opening of multiple tabs with predefined URLs.

For more information on Brave plugin and its capabilities, refer to the [Brave plugin documentation](https://anyquery.dev/integrations/brave).
