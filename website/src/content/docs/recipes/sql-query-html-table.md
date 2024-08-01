---
title: Query an HTML table
description: Learn how to run a SQL query on an HTML table from a website     
---

Anyquery is a SQL query engine that enables you to execute SQL queries on virtually anything, including HTML tables. In this recipe, we will run a SQL query on [diskprices.com](https://diskprices.com) to find the cheapest 12TB HDD.

The `read_html` table function takes a URL and a CSS selector that points to an HTML table. The following query selects the cheapest 1TB SSD from [diskprices.com](https://diskprices.com) and outputs the result in JSON format:

```bash
anyquery -q "SELECT * FROM read_html('https://diskprices.com', 'table') WHERE Technology = 'HDD' AND Capacity = '12 TB' ORDER BY Price LIMIT 1" --json
```
