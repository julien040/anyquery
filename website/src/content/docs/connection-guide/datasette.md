---
title: Datasette
description: Connect Datasette to Anyquery
---

<img src="https://datasette.io/static/datasette-card.png" alt="Datasette" width="400"/>

Datasette makes it easy to explore and publish data. Anyquery makes it easy to query data from any source. I see a match here. Let's connect Datasette to Anyquery.

## Prerequisites

Before you begin, ensure that you have the following:

- A working installation of Anyquery
- Datasette installed on your machine

## Fetching Data

For the example, we will import two datasets:

- Commits from [simow/datasette](https://github.com/simonw/datasette/)
- [Foreign exchange rates](https://csvbase.com/table-munger/eurofxref) from CSVBase

Let's begin by installing the `git` plugin if it has not been done already:

```bash
anyquery install git
```

Next, let's open a shell with an on-disk database and import the data:

```bash
anyquery q datasette.db
```

```sql
-- Importing commits from simonw/datasette
CREATE TABLE datasette_commits AS
SELECT * FROM git_commits_diff('https://github.com/simonw/datasette.git');

-- Importing foreign exchange rates
CREATE TABLE euro_exchange_rates AS
SELECT * FROM read_parquet('https://csvbase.com/table-munger/eurofxref.parquet', header=true);

-- Vacuum the database to be sure all data is in datasette.db
VACUUM;
```

## Connecting Datasette

Let's write the Datasette metadata file. It adds a title, description, and a custom theme created by [julien040](https://github.com/julien040/charcoal-datasette-theme).

```json
// metadata.json
{
    "title": "My datasette - anyquery integration",
    "description": "This is a datasette instance connected to anyquery",

    "extra_css_urls": [
        "https://cdn.jsdelivr.net/gh/julien040/charcoal-datasette-theme@1.0.0/theme.min.css"
    ],
}
```

Now, let's start Datasette:

```bash
datasette datasette.db --metadata metadata.json
```

Head to [http://127.0.0.1:8001/datasette/datasette_commits](http://127.0.0.1:8001/datasette/datasette_commits) to see the commits from the datasette repository.

![Datasette commits](/images/docs/tR5IawXS.png)

## Conclusion

You've successfully connected Datasette to Anyquery. You can see the result at [https://anyquery-datasette-example.anyquery.dev](https://anyquery-datasette-example.anyquery.dev)
