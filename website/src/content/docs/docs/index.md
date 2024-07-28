---
title: Getting Started
description: Learn more about what Anyquery is and how to use it
---

<img src="/images/docs-header.svg" alt="stars" />

## What is anyquery about ?

Anyquery allows you to write SQL queries on pretty much any data source. It is a query engine that can be used to query data from different sources like databases, APIs, and even files. For example, you can use a Notion database or a Google Sheet as a database to store your data.

**Example**

```sql
-- List all the repositories from Cloudflare ordered by stars
SELECT * FROM github_repositories_from_user('cloudflare') ORDER BY stargazers_count DESC;

-- List all your saved tracks from Spotify
SELECT * FROM spotify_saved_tracks;

-- Insert data from a git repository into a Google Sheet
INSERT INTO google_sheets_table (name, line_added) SELECT author_name, addition FROM git_commits_diff('https://github.com/vercel/next.js.git');
```

## Installation

TODO: Installation instructions
