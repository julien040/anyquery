---
title: Welcome
description: Learn more about what Anyquery is and how to use it
template: doc
hero:
    actions:
    - text: Tell me more
      link: /get-started/
      icon: right-arrow
      variant: primary
    - text: View on GitHub
      link: https://github.com/julien040/anyquery
      icon: external
    title: 'Anyquery'
    image: 
        html: '<img src="/images/docs-header.svg" alt="stars" />'
    
        
---

## What is anyquery about ?

Anyquery allows you to write SQL queries on pretty much any data source. It is a query engine that can be used to query data from different sources like databases, APIs, and even files. For example, you can use a Notion database or a Google Sheet as a database to store your data.

### Example

Let's use the [Google Sheets plugin](/integrations/google_sheets) to query data from a Google Sheet. Once you have followed the instructions to connect your Google Sheet, you can run the following queries. Anyquery will infer the schema of the Google Sheet and allow you to query it using SQL.

Open a new SQL shell by running the following command:

```bash
anyquery
```

Now, you can run the following queries:

```sql
-- Observe the infered schema
DESCRIBE google_sheets_table;
-- Query data from a Google Sheet
SELECT * FROM google_sheets_table;
-- Insert a new row in a Google Sheet
INSERT INTO google_sheets_table (name, age) VALUES ('John', 25);
```

### Why use Anyquery ?

#### As a import/export tool

