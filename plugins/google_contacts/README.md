# Google Contacts plugin

This plugin allows you to query your Google Contacts using SQL queries.

## Setup

Install the plugin with:

```bash
anyquery install google_contacts
```

Then, you need to authenticate with Google. Go to the [Google Cloud Console](https://console.cloud.google.com/), create a new project, and go to the [APIs & Services console](https://console.cloud.google.com/apis/dashboard).

1. Click on Credentials
    ![alt text](https://cdn.jsdelivr.net/gh/julien040/anyquery@main/plugins/google_forms/images/identifier.png)
2. Click on Create Credentials, and select OAuth client ID
    ![alt text](https://cdn.jsdelivr.net/gh/julien040/anyquery@main/plugins/google_forms/images/create.png)
3. If not done, configure the consent screen
    ![alt text](https://cdn.jsdelivr.net/gh/julien040/anyquery@main/plugins/google_forms/images/consentScreen.png)
    1. And fill the form with the required information
        - Application name: AnyQuery
        - User support email: Your email
        - Developer contact information: Your email
        - Leave the rest as it is

        ![alt text](https://cdn.jsdelivr.net/gh/julien040/anyquery@main/plugins/google_forms/images/consentFilled.png)
    2. Click on Save and Continue
    3. Click on Save and Continue and leave Scopes as it is
    4. On test users, add the Google account you will use to query the responses
    5. Click on Save and Continue
    6. Click on Back to Dashboard
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
8. Enable the Google People API. To do so, go to the [Google People API page](https://console.cloud.google.com/apis/library/people.googleapis.com) and click on Enable
9. Go to [Google Contacts integration](https://integration.anyquery.dev/google-contacts)
10. Fill the form with the `Client ID` and `Client Secret` you copied and click on Submit
    ![alt text](https://cdn.jsdelivr.net/gh/julien040/anyquery@main/plugins/google_forms/images/form_integration.png)
11. Select your Google account, skip the warning about the app not being verified, and
12. Copy the token, the client ID, and the client secret
    ![alt text](https://cdn.jsdelivr.net/gh/julien040/anyquery@main/plugins/google_forms/images/token.png)
13. Go back to the terminal and fill in the form with the token, the client ID, and the client secret.

When `anyquery` finishes the installation, you will be asked to provide the token, the client ID, and the client Secret. Once you have provided the information, the plugin will be ready to use.

## Usage

You can query your Google Contacts using SQL queries. The table name are `google_contacts_items` and `google_contacts_flat`. The former returns each value of the field as a JSON array or object, while the latter returns each first value of the field.

Here are some examples:

```sql
-- List your 10 first contacts by order of creation
SELECT * FROM google_contacts_flat LIMIT 10;
-- List contacts that will have their birthday in the next 90 days
SELECT names, substring(birthdays, 6, 5) AS birthday
FROM google_contacts_flat WHERE birthday BETWEEN strftime('%m-%d', 'now') AND strftime('%m-%d', 'now', '+90 days');
-- List contacts that have a phone number
SELECT names, phone_numbers FROM google_contacts_flat WHERE phone_numbers IS NOT NULL;
-- List all email addresses of contacts
SELECT names ->> '$[0]' as name, a.value AS email FROM google_contacts_items, json_each(email_addresses) AS a;
```

## Schema

The schema is the same for both tables `google_contacts_items` and `google_contacts_flat`. The main difference is that `google_contacts_items` returns JSON values. For examples, `names` will be `["John Doe"]` in `google_contacts_items` and `John Doe` in `google_contacts_flat`. However, in case of multiple values, `google_contacts_items` will return an array of values, while `google_contacts_flat` will return the first value.

Here is the schema:

| Column index | Column name     | type |
| ------------ | --------------- | ---- |
| 0            | id              | TEXT |
| 1            | addresses       | TEXT |
| 2            | age_range       | TEXT |
| 3            | biographies     | TEXT |
| 4            | birthdays       | TEXT |
| 5            | calendar_urls   | TEXT |
| 6            | client_data     | TEXT |
| 7            | cover_photos    | TEXT |
| 8            | email_addresses | TEXT |
| 9            | events          | TEXT |
| 10           | gender          | TEXT |
| 11           | im_clients      | TEXT |
| 12           | interests       | TEXT |
| 13           | locales         | TEXT |
| 14           | locations       | TEXT |
| 15           | names           | TEXT |
| 16           | nicknames       | TEXT |
| 17           | occupations     | TEXT |
| 18           | organizations   | TEXT |
| 19           | phone_numbers   | TEXT |
| 20           | photos          | TEXT |
| 21           | relations       | TEXT |
| 22           | sip_addresses   | TEXT |
| 23           | skills          | TEXT |
| 24           | urls            | TEXT |
| 25           | user_defined    | TEXT |

## Limitation

- The plugin does not do any caching. It will query the Google Contacts API each time you run a query. While the rate limits of the API are generous, you may hit them if you run too many queries in a short period. <br>To circumvent this limitation, you can save your query results in a table and query this table instead of the Google Contacts API. For example, run `CREATE TABLE my_contacts AS SELECT * FROM google_contacts_flat;` and then query `my_contacts` instead of `google_contacts_flat`.
- The plugin does not support the `INSERT`, `UPDATE`, and `DELETE` statements. It is planned to support them in the future. If you need to modify your contacts with SQL, open an issue, and I will prioritize it.
