# Google Forms plugin

Query the responses of a Google Form with SQL.

## Usage

Each Google Form response is a row in the `google_forms_responses` table. The columns are the questions of the form, and the values are the responses.

```sql
-- Get all the responses
SELECT * FROM google_forms_responses;

-- Get the first choice of a multiple-choice question
SELECT my_question->>'$[0]' FROM google_forms_responses;
```

### Multiple forms

Anyquery can handle multiple profiles. To do so, you need to create a new profile for each form you want to analyze.

```bash
# Create a new profile and follow the instructions of the plugin (it will ask for the token, the client ID, the client secret, and the form ID)
anyquery profile new default google_forms mycustomname
# Once done, you can query your new profile
anyquery -q "SELECT * FROM mycustomname_google_forms_responses"
```

## Setup

Install the plugin with:

```bash
anyquery install google_forms
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
8. Enable the Google Forms API. To do so, go to the [Google Forms API page](https://console.cloud.google.com/apis/library/forms.googleapis.com) and click on Enable
    ![alt text](https://cdn.jsdelivr.net/gh/julien040/anyquery@main/plugins/google_forms/images/enableAPI.png)
9. Go to [Google Forms integration](https://integration.anyquery.dev/google-forms)
10. Fill the form with the `Client ID` and `Client Secret` you copied and click on Submit
    ![alt text](https://cdn.jsdelivr.net/gh/julien040/anyquery@main/plugins/google_forms/images/form_integration.png)
11. Select your Google account, skip the warning about the app not being verified, and
12. Copy the token, the client ID, and the client secret
    ![alt text](https://cdn.jsdelivr.net/gh/julien040/anyquery@main/plugins/google_forms/images/token.png)
13. Go back to the terminal and fill in the form with the token, the client ID, and the client secret.
14. To find the form ID, go to the form edit page and copy the ID from the URL
   In `https://docs.google.com/forms/d/5aP_uDmwzxXQF_nOlrkFD1tji97brPhLQ6NLZRnz4E80/edit`, the form ID is `5aP_uDmwzxXQF_nOlrkFD1tji97brPhLQ6NLZRnz4E80`

## Schema

| Column index | Column name | type |
| ------------ | ----------- | ---- |
| 0            | id          | TEXT |
| 1            | created_at  | TEXT |

The rest of the columns are the questions of the form.

## Notes

- The plugin can only query the responses of a form that you have access to. It cannot insert, update, or delete responses.
- The plugin does not do any caching. It queries the responses every time you run a query. To speed up the queries, you can save the results to a SQLite database with the `CREATE TABLE my_table AS SELECT * FROM google_forms_responses` query.
- As always with `anyquery`, arrays are json-encoded. If you want to query them, you can use the `->>` operator to extract the value. For example, to get the first choice of a multiple-choice question, you can use `SELECT my_question->>'$[0]' FROM google_forms_responses`.
