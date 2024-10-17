# Salesforce

The Salesforce custom plugin provides a way to interact with a Salesforce sObject using SQL queries. It allows you to query, insert, update, and delete records from Salesforce objects.

Compared to the [Salesforce plugin](https://anyquery.dev/integrations/salesforce), the Salesforce custom plugin doesn't have a predefined list of tables. Instead, you input the sobject you want to query, and the plugin will make the sObject available as a table in the query.

## Usage

In the plugin configuration, you will provide the sobject you want to query. Then, the plugin will make the sObject available as a table in the query.

```sql
-- Let's say you chose the account sobject
SELECT Id, Name FROM salesforce_custom_object

-- Now you have a second profile named contact with the contact sobject configured
SELECT Id, LastName FROM contact_salesforce_custom_object
```

If you want to query across multiple organizations, you can [create multiple profiles](https://anyquery.dev/docs/usage/managing-profiles) and configure each profile with the credentials of a different organization.

```bash
anyquery profile new default salesforce_custom mysobject
```

## Configuration

To get started, install the plugin:

```bash
anyquery install salesforce_custom
```

To use this plugin, you will need a Salesforce organization with the REST API enabled. Currently, there are four ways to authenticate with the Salesforce plugin:

- Username and Password
- Access Token using the `sf` CLI
- OAuth 2.0 Web Server Flow
- OAuth 2.0 Client Credentials Flow

All flows require the following configuration:

- `domain`: The domain of your Salesforce organization. For example, `mydomain.my.salesforce.com`. You need to left out the `https://` part. You can find this in the URL of the login page. <br> In `https://mycompany.lightning.force.com/lightning/setup`, your domain is `mycompany.my.salesforce.com`. For more information, see [Finding Your Salesforce Domain](https://help.salesforce.com/s/articleView?id=sf.faq_domain_name_what.htm&type=5).
- `cache_ttl`: The time-to-live (TTL) for the cache in seconds. The default is 0, which means no cache.
- `encryption_key`: The encryption key used to encrypt the cache for sensitive data. The key must either be 16, 24, or 32 bytes long. Make sure you are using ascii characters only. If you don't provide an encryption key, the plugin will not be able to start.

### Access Token using the `sf` CLI

1. Install the `sf` CLI by [following the instructions](https://developer.salesforce.com/docs/atlas.en-us.sfdx_setup.meta/sfdx_setup/sfdx_setup_install_cli.htm)
2. Run `sf org login web` to authenticate with Salesforce
3. Run `sf org display --target-org <username>` and replace `<username>` with your username
4. Copy the Access Token from the output

When anyquery will ask for the access token, paste the token you copied. Fill in the `domain`, `encryption_key`, and `cache_ttl` in the configuration, and leave the other fields empty.

Note that the access token expires after a certain amount of time (usually 1 hour). You can refresh the access token by running again `sf org login web` and copying the new access token. Then run `anyquery profiles update default salesforce_custom <profile name>` (*default* is the main profile name) and paste the new access token in the access token field. Leave the other fields empty.

### OAuth 2.0 Web Server Flow

#### Step 1: Create a Connected App

1. Go to `Setup` > `App Manager` > `New Connected App`
2. Fill in the required fields
3. In the `API (Enable OAuth Settings)` section, check `Enable OAuth Settings`
4. In the `Callback URL` field, enter `http://localhost:8080/callback`
5. In the `Selected OAuth Scopes` section, add the required scopes
    - Access and manage your data (api)
    - Perform requests on your behalf at any time (refresh_token, offline_access)
6. Click `Save`

#### Step 2: Get the Consumer Key and Consumer Secret

1. Go to `Setup` > `App Manager` > `Manage Connected Apps`
2. Click on the app you created
3. Open OAuth Settings accordion
4. Click on `Consumer Key`, pass the security check, and copy the consumer key and consumer secret

#### Step 3: Authenticate with Salesforce

Open in your browser the following URL and replace the placeholders with your values:

```url
https://<domain>/services/oauth2/authorize?response_type=code&client_id=<consumer_key>&redirect_uri=http://localhost:8080/callback
```

After you authenticate, you will be redirected to `http://localhost:8080/callback?code=<code>`. Copy the code from the URL.

#### Step 4: Get the refresh token

Run the following command in your terminal and replace the placeholders with your values:

```bash
curl -X "POST" "https://<domain>/services/oauth2/token" \
     -H 'Content-Type: application/x-www-form-urlencoded; charset=utf-8' \
     --data-urlencode "grant_type=authorization_code" \
     --data-urlencode "code=<the code you copied from the URL>" \
     --data-urlencode "client_id=<consumer_key>" \
     --data-urlencode "client_secret=<consumer_secret>" \
     --data-urlencode "redirect_uri=http://localhost:8080/callback"
```

Copy the `refresh_token` from the response. When anyquery will ask for the refresh token, paste the token you copied. Fill in the `domain`, `consumer_key`, `consumer_secret`, `encryption_key`, and `cache_ttl` in the configuration, and leave the other fields empty.

### OAuth 2.0 Client Credentials Flow

#### Step 1: Create a Connected App

1. Go to `Setup` > `App Manager` > `New Connected App`
2. Fill in the required fields
3. In the `API (Enable OAuth Settings)` section, check `Enable OAuth Settings`
4. In the `Callback URL` field, enter whatever you want
5. In the `Selected OAuth Scopes` section, add the required scopes
    - Access and manage your data (api)
    - Perform requests on your behalf at any time (refresh_token, offline_access)
6. Click `Save`

Now you need to enable the `Client Credentials OAuth Flow` for the connected app:

1. Go to `Setup` > `App Manager` > `Manage Connected Apps`
2. Click on the app you created
3. Click on `Edit Policies`
4. Under OAuth Policies, select "Enable Client Credentials OAuth Flow"
5. In the input field just below, enter the email adress of the user you want to impersonate for Anyquery
6. Click `Save`

#### Step 2: Get the Consumer Key and Consumer Secret

1. Go to `Setup` > `App Manager` > `Manage Connected Apps`
2. Click on the app you created
3. Open OAuth Settings accordion
4. Click on `Consumer Key`, pass the security check, and copy the consumer key and consumer secret

When asked by Anyquery, paste the consumer key and consumer secret. Fill in the `domain`, `encryption_key`, and `cache_ttl` in the configuration, and leave the other fields empty.

### Username and Password

This flow is not recommended for security reasons. If you can, prefer the Client Credentials Flow.

#### Step 1: Create a Connected App

1. Go to `Setup` > `App Manager` > `New Connected App`
2. Fill in the required fields
3. In the `API (Enable OAuth Settings)` section, check `Enable OAuth Settings`
4. In the `Callback URL` field, enter `http://localhost:8080/callback`
5. In the `Selected OAuth Scopes` section, add the required scopes
    - Access and manage your data (api)
    - Perform requests on your behalf at any time (refresh_token, offline_access)
6. Click `Save`

#### Step 2: Enable the Username and Password OAuth Flow

1. Go to `Setup` > `Identity` > `OAuth and OpenID Connect Settings`
2. Check `Allow users to use the username/password OAuth flow` and click `Save`

Once done, when asked by Anyquery, paste the consumer key (client_id) and consumer secret (client_secret). Fill in the `domain`, `encryption_key`, and `cache_ttl` in the configuration.
You'll need to enter your username and password when Anyquery asks for them. Your password is the concatenation of your password and your security token. Learn how to get your security token [here](https://help.salesforce.com/s/articleView?id=xcloud.user_security_token.htm&type=5).

## Additional Information

- The plugin uses the Salesforce REST API to interact with Salesforce. As a rule of thumb, querying 2000 records costs 1 API request. And 200 Insert, Update, or Delete operations cost 1 API request. Refer to the [Salesforce documentation](https://help.salesforce.com/s/articleView?id=000389027&type=1) to find your monthly API request limit.
- Queries using `sum`, `count` or other aggregate functions may result in a lot of API requests because Anyquery has to fetch all the records to calculate the result. Avoid them on `sobject` that have a lot of records.
- The plugin uses the v61.0 version (Summer '24) of the Salesforce API.
- The plugin caches the results of the queries to reduce the number of API requests. The cache is encrypted using the encryption key provided in the configuration. You can change the cache TTL in the configuration.
- Internally, the plugin transforms the SQL queries into SOQL queries to interact with Salesforce.
- Any insert or update on fields that are not updatable will be ignored.
- The plugin doesn't support the Bulk API. If it's a requirement for you, feel free to [open an issue](https://github.com/julien040/anyquery/issues/new)

## Schema

The schema will depend of the object you are querying.
