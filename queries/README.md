# Queries template

![Community queries](./community-repo.png)

This directory contains common SQL queries that can be used to query data from diverse sources. Head to [https://anyquery.dev/queries](https://anyquery.dev/queries) to see the full list of queries.

Once you find a query you like, you can run

```bash
anyquery run <query-name-without-extension>
```

and Anyquery will take care of installing the necessary dependencies, asking you the necessary parameters, and running the query for you.

Moreover, you can export the query results to different formats by passing the `--format` flag (.e.g. `--format csv`, `--format json`, `--format markdown`, etc.). See [Anyquery documentation](https://anyquery.dev/docs/usage/exporting-results/) for more information.

If you have a query that you would like to share with the community, please submit a PR to this repository. Below is explained how to do it.

## How to submit a query

1. Fork this repository.
2. Create a file `.sql` with the query you want to share. The file should be placed in the `queries` directory.
3. Create a pull request to this repository.
4. Once the PR is merged, the query will be available at [https://anyquery.dev/queries](https://anyquery.dev/queries).

If you have issues with pull requests, feel free to send the query to [mailto:contact@anyquery.dev](contact@anyquery.dev) and I will take care of adding it to the repository.

### Query template

```sql
/*
 title = "GitHub Stars per day"
 description = "Discover the number of stars per day for a given repository. This query returns the top 10 days with the most stars."
 
 plugins = ["github"]

 arguments = [
    {title="repository", display_title = "Repository name (owner/repo format)", type="string", description="The repository to fetch stars from (owner/repo)", regex="^[a-zA-Z0-9_-]+/[a-zA-Z0-9_-]+$"}
 ]
 
 */
SELECT date(starred_at) as day, count(*) as stars FROM github_stargazers_from_repository(@repository) GROUP BY day ORDER BY stars DESC;
```

Each query starts will a multi-line comment that contains metadata about the query. The metadata is used by Anyquery to display the query in the UI and to ask the user for the necessary parameters. The metadata is a TOML-like format composed of the following fields:

- `title`: The title of the query.
- `description`: A brief description of the query.
- `plugins`: The plugins required to run the query. In this case, the query requires the `github` plugin.
- `arguments`: The arguments required by the query. Each argument is an object with the following fields:
  - title: The name of the variable.
  - display_title: The name of the variable as it will be displayed to the user.
  - type: The type of the variable. It can be `string`, `int`, `float`, `bool`
  - description: A brief description of the variable to help the user in the form.
  - regex: A regular expression to validate the input. Optional.

After the metadata, the query is written in plain SQL. The query can contain placeholders that will be replaced by the values provided by the user. In this case, the query uses the `@repository` placeholder that will be replaced by the value provided by the user for the argument `repository`.

You can have multiple queries in the same file. Just make sure to separate them with a blank line for dot commands, and a semicolon for SQL queries.

> ⚠️ Dot queries
>
> Dot queries cannot have comments in the same line. This means you cannot have a dot comment right after a comment (even the metadata comment). To run a dot query, make sure to have a SQL query before it or add them at the top of the file.

You can test the query by running

```bash
anyquery run <path-to-query-file-with-extension>
```
