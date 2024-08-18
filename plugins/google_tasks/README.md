# Google Tasks plugin

The Google Tasks plugin allows you to interact with your Google Tasks lists and tasks.

## Setup

Install the plugin with:

```bash
anyquery install google_tasks
```

Then, you need to authenticate with Google. Go to the [Google Cloud Console](https://console.cloud.google.com/), create a new project, and go to the [APIs & Services console](https://console.cloud.google.com/apis/dashboard).

1. Click on Credentials
    ![alt text](https://cdn.jsdelivr.net/gh/julien040/anyquery@main/plugins/google_forms/images/identifier.png)
2. Click on Create Credentials, and select OAuth client ID
    ![alt text](https://cdn.jsdelivr.net/gh/julien040/anyquery@main/plugins/google_forms/images/create.png)
3. If not done, configure the consent screen
    ![alt text](https://cdn.jsdelivr.net/gh/julien040/anyquery@main/plugins/google_forms/images/consentScreen.png)
    1. Select `External` and click on Create
    2. And fill the form with the required information
        - Application name: AnyQuery
        - User support email: Your email
        - Developer contact information: Your email
        - Leave the rest as it is

        ![alt text](https://cdn.jsdelivr.net/gh/julien040/anyquery@main/plugins/google_forms/images/consentFilled.png)
    3. Click on Save and Continue
    4. Click on Save and Continue and leave Scopes as it is
    5. On test users, add the Google account you will use to query the responses
    6. Click on Save and Continue
    7. Click on Back to Dashboard
4. Go back to the Credentials tab and click on Create Credentials
    ![alt text](https://cdn.jsdelivr.net/gh/julien040/anyquery@main/plugins/google_forms/images/createCredentials.png)
5. Select OAuth client ID, and select Web application
6. Fill the form with the required information
    - Leave the name as whatever you want
    - Add the authorized redirect URIs: `https://integration.anyquery.dev/google-result`
    - Add authorized JavaScript origins: `https://integration.anyquery.dev`
    - Click on Create
    ![alt text](https://cdn.jsdelivr.net/gh/julien040/anyquery@main/plugins/google_forms/images/form_oAuth.png)
7. Copy the `Client ID` and `Client Secret`. We will use them later
    ![alt text](https://cdn.jsdelivr.net/gh/julien040/anyquery@main/plugins/google_forms/images/result.png)
8. Enable the Google Tasks API. To do so, go to the [Google Tasks API page](https://console.cloud.google.com/apis/library/tasks.googleapis.com) and click on Enable
9. Go to [Google Tasks integration](https://integration.anyquery.dev/google-tasks)
10. Fill the form with the `Client ID` and `Client Secret` you copied and click on Submit
    ![alt text](https://cdn.jsdelivr.net/gh/julien040/anyquery@main/plugins/google_forms/images/form_integration.png)
11. Select your Google account, skip the warning about the app not being verified, and
12. Copy the token, the client ID, and the client secret
    ![alt text](https://cdn.jsdelivr.net/gh/julien040/anyquery@main/plugins/google_forms/images/token.png)
13. Go back to the terminal and fill in the form with the token, the client ID, and the client secret.

When `anyquery` finishes the installation, you will be asked to provide the token, the client ID, and the client Secret. Once you have provided the information, the plugin will be ready to use.

## Usage

You can now query your Google Tasks using SQL queries. Most of the time, you'll be requested the list id. You can find it by listing all your tasks lists with the following query:

```sql
SELECT * FROM google_tasks_lists;
```

The id is the first column of the result. Now, here are some examples of queries you can run with your Google Tasks:

```sql
-- List all tasks
SELECT * FROM google_tasks_items('list-id');
SELECT * FROM google_tasks_items WHERE list_id = 'list-id';
-- List all tasks, even the deleted ones
SELECT * FROM google_tasks_items('list-id', TRUE);
-- Set to completed all tasks that have the word 'done' in the title
UPDATE google_tasks_items SET status = 'completed', completed_at = '2024-08-15' WHERE title LIKE '%done%' and list_id = 'list-id';
-- Insert a new task
INSERT INTO google_tasks_items (list_id, title, due_at) VALUES ('list-id', 'New task', '2021-12-31T20:15:00Z');
```

## Schema

### google_tasks_items

List all tasks of the specified list (`list_id` parameter)

| Column index | Column name  | type    |
| ------------ | ------------ | ------- |
| 0            | id           | TEXT    |
| 1            | title        | TEXT    |
| 2            | status       | TEXT    |
| 3            | completed_at | TEXT    |
| 4            | due_at       | TEXT    |
| 5            | updated_at   | TEXT    |
| 6            | links        | TEXT    |
| 7            | notes        | TEXT    |
| 8            | parent_id    | TEXT    |
| 9            | position     | TEXT    |
| 10           | url          | TEXT    |
| 11           | hidden       | INTEGER |
| 12           | deleted      | INTEGER |

### google_tasks_lists

List all tasks lists

| Column index | Column name | type |
| ------------ | ----------- | ---- |
| 0            | id          | TEXT |
| 1            | title       | TEXT |
| 2            | updated_at  | TEXT |

## Known limitations

- The plugin does not do any caching. Each query will fetch the data from Google Tasks. The default allowance is 50,000 queries per day. It should be enough for most use cases. If you have a lot of tasks and you only want to run analytics queries, consider importing the data into a database. To do so, run `CREATE TABLE google_tasks_items AS SELECT * FROM google_tasks_items('list-id')`.
- You cannot delete a task using SQL queries. You can only mark it as completed.
- Tasks list can only be queried to retrieve their id. You cannot INSERT/UPDATE/DELETE tasks lists. If this feature is important to you, please open an issue to prioritize it.

## Troubleshooting

### SERVICE_DISABLED

If you have an issue similar to this:

```error
unable to list task lists: googleapi: Error 403: Google Tasks API has not been used in project before or it is disabled. Enable it by visiting https://console.developers.google.com/apis/api/tasks.googleapis.com/overview then retry. If you enabled this API recently, wait a few minutes for the action to propagate to our systems and retry.
accessNotConfigured

SERVICE_DISABLED
```

make sure you have enabled the Google Tasks API in the Google Cloud Console.
