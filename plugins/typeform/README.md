# Typeform plugin

Analyse the results of a Typeform survey with SQL.

## Setup

1. Go to the [Typeform Personal token page](https://admin.typeform.com/user/tokens) and create a new token (Click on "Generate a new token").
2. Select the following scopes (leaving the others checked is a non-issue. Yet, it might put your data at risk):
   - Account: Read
   - Forms: Read
   - Responses: Read
  ![Scope](https://cdn.jsdelivr.net/gh/julien040/anyquery@main/plugins/typeform/images/scopes.png)
3. Copy the token and paste it when asked for it in the plugin.
4. Go to the [Typeform forms page](https://admin.typeform.com/), click on the form you want to analyze, and copy the form ID from the URL.
On the URL `https://admin.typeform.com/form/tabc4poi/create?block=c89e2e8f-e3e3-420c-9ba6-21d1ad0a7bf4`, the form ID is `tabc4poi`.
5. Paste the form ID when asked for it in the plugin.

### Multiple forms

Anyquery can handle multiple profiles. To do so, you need to create a new profile for each form you want to analyze.

```bash
# Create a new profile and follow the instructions of the plugin (it will ask for the token and the form ID)
anyquery profile new default typeform mycustomname
# Once done, you can query your new profile
anyquery -q "SELECT * FROM mycustomname_typeform_responses"
```

## Example

The schema changes depending on the form, but the plugin will create a table with the name `typeform_responses` that contains all the responses to the form.

```sql
-- Get all the responses
SELECT * FROM typeform_responses;

-- Get the number of responses
SELECT COUNT(*) FROM typeform_responses;

-- Get the count of submissions per day
SELECT count(*) as "Count of submissions", date(submitted_at) as Date FROM typeform_responses GROUP BY date(submitted_at);

-- Save all the results to a SQLite database in order to speed up the queries by avoiding the API calls
CREATE TABLE my_table AS SELECT * FROM typeform_responses;
SELECT * FROM my_table;
```

```bash
# Export the results to a CSV file
anyquery -q "SELECT * FROM typeform_responses" --csv > responses.csv
# Export the results to a JSON file
anyquery -q "SELECT * FROM typeform_responses" --json > responses.json
# Export the results to a SQLite database
anyquery q my.db -q "CREATE TABLE my_table AS SELECT * FROM typeform_responses"
```

## Schema

The schema of the responses table is dynamic and depends on the form. The plugin will create a table with the name `typeform_responses` that contains all the responses to the form.
The following columns are common to all the forms:

| Column index | Column name   | type |
| ------------ | ------------- | ---- |
| 0            | id            | TEXT |
| 1            | landed_at     | TEXT |
| 2            | submitted_at  | TEXT |
| 3            | user_agent    | TEXT |
| 4            | response_type | TEXT |

The rest of the columns are the answers to the form questions. The column name is the question title, appended by an underscore if the title already exists in the table. Subfield titles (like an address) are composed of the question title and the subfield title, separated by an underscore.

## Notes

- The plugin does not do any caching, so it will make a request to the Typeform API every time you run a query.
At the time of writing, the Typeform API has a rate limit of 2 requests per second. If you hit the rate limit, the plugin will wait for a second before making the next request. This means you can't export more than 2000 responses per second.
- To simulate caching, you can save the results to a SQLite database and query the database instead of the API (example above).
