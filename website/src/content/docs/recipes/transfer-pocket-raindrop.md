---
title: Transfer your Pocket articles to Raindrop.io
description: Learn how to transfer your Pocket articles to Raindrop.io using Anyquery.
---

[Pocket](https://getpocket.com/) is a read-it-later service that allows you to save articles, videos, and other content to read later. [Raindrop.io](https://raindrop.io/) is a bookmark manager that enables you to organize your bookmarks, articles, and other content in one place.

Anyquery is a SQL query engine that enables you to execute SQL queries on virtually anything, including Pocket and Raindrop.io. In this recipe, we will transfer your Pocket articles to Raindrop.io using Anyquery.

## Setup

Before you can transfer your Pocket articles to Raindrop.io, you need to install Anyquery and authenticate with Pocket and Raindrop.io.
To do this, follow the steps below:

- Install Anyquery by following the [installation instructions](/docs/#installation).
- Set up the Pocket integration by following the [Pocket authentication instructions](/integrations/pocket).
- Set up the Raindrop.io integration by following the [Raindrop.io authentication instructions](/integrations/raindrop).

Once done, check that both connections are working by running the following commands:

```bash
anyquery -q "SELECT * FROM pocket_items LIMIT 1; SELECT * FROM raindrop_items LIMIT 1"
```

## Transfer your Pocket articles to Raindrop.io

To transfer your Pocket articles to Raindrop.io, you can use the following SQL query:

```bash
anyquery -q "INSERT INTO raindrop_items (title, link) SELECT resolved_title as title, resolved_url as link FROM pocket_items"
```

If you have a collectionID in Raindrop.io where you want to save the articles, you can specify it in the query:

```bash
anyquery -q "INSERT INTO raindrop_items (title, link, collection_id) SELECT resolved_title as title, resolved_url as link, your-collection-id-integer as collection_id FROM pocket_items"
```

:::tip
You can insert up to 12,000 items per minute due to the Raindrop.io API rate limit. No worries, anyquery will automatically wait for the next minute if you exceed the limit.
:::
