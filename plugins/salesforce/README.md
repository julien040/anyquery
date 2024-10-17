# Salesforce

The Salesforce plugin provides a way to interact with Salesforce using SQL. You can query Salesforce objects, create new Salesforce objects, update Salesforce objects, and delete Salesforce objects.

## Usage

The Salesforce plugin provides tables for the following objects: `account`, `contact`, `lead`, `opportunity`, `case`, `task`, `event`, `campaign`, `user`, `campaignmember`, `asset`, `contract`, `contractlineitem`, `servicecontract`, `solution`, `pricebook2`, `product2`, `productitem`, `pricebookentry`, `quote`, `quotelineitem`, `order`, `orderitem`, `invoice`, `invoiceline`, `report`, `dashboard`, `document`, `payment`, `paymentlineinvoice`

If you need to query a different object, you can use the `salesforce_custom` plugin available [here](https://anyquery.dev/integrations/salesforce_custom).

```sql

-- Query all accounts
SELECT * FROM salesforce_account;

-- Query all contacts whose email is like 'john.doe%'
SELECT * FROM salesforce_contact WHERE email LIKE 'john.doe%';

-- Join accounts and contacts
SELECT a.name, c.email FROM salesforce_account a
JOIN salesforce_contact c ON a.id = c.accountid;

-- Create a new account
INSERT INTO salesforce_account (name, phone) VALUES ('My Account', '1234567890');

-- Update an account
UPDATE salesforce_account SET phone = '0987654321' WHERE name = 'My Account';

-- Delete an account
DELETE FROM salesforce_account WHERE name = 'My Account';

-- Sum of the amount charged in all invoices
SELECT SUM(TotalChargeAmount) FROM salesforce_invoice;

```

If you want to query across multiple organizations, you can [create multiple profiles](https://anyquery.dev/docs/usage/managing-profiles) and configure each profile with the credentials of a different organization.

## Configuration

To get started, install the plugin:

```bash
anyquery install salesforce
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

Note that the access token expires after a certain amount of time (usually 1 hour). You can refresh the access token by running again `sf org login web` and copying the new access token. Then run `anyquery profiles update default salesforce <profile name>` (*default* is the main profile name) and paste the new access token in the access token field. Leave the other fields empty.

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
- The plugin doesn't supported `sobjectFeed` object. This means you cannot access the feed of an object like `accountFeed`, `contactFeed`, etc. As a workaround, you can use the `salesforce_custom` plugin to query the feed of an object.

## Schema

The following contains the schema of the tables available in the Salesforce plugin. If you added any custom fields to your Salesforce objects, they will be available in the tables suffixed with `__c`.

### salesforce_account

Represents an individual account, which is an organization or person involved with your business (such as customers, competitors, and partners).

| Column index | Column name             | type    |
| ------------ | ----------------------- | ------- |
| 0            | Id                      | TEXT    |
| 1            | IsDeleted               | INTEGER |
| 2            | MasterRecordId          | TEXT    |
| 3            | Name                    | TEXT    |
| 4            | Type                    | TEXT    |
| 5            | ParentId                | TEXT    |
| 6            | BillingStreet           | TEXT    |
| 7            | BillingCity             | TEXT    |
| 8            | BillingState            | TEXT    |
| 9            | BillingPostalCode       | TEXT    |
| 10           | BillingCountry          | TEXT    |
| 11           | BillingLatitude         | REAL    |
| 12           | BillingLongitude        | REAL    |
| 13           | BillingGeocodeAccuracy  | TEXT    |
| 14           | BillingAddress          | TEXT    |
| 15           | ShippingStreet          | TEXT    |
| 16           | ShippingCity            | TEXT    |
| 17           | ShippingState           | TEXT    |
| 18           | ShippingPostalCode      | TEXT    |
| 19           | ShippingCountry         | TEXT    |
| 20           | ShippingLatitude        | REAL    |
| 21           | ShippingLongitude       | REAL    |
| 22           | ShippingGeocodeAccuracy | TEXT    |
| 23           | ShippingAddress         | TEXT    |
| 24           | Phone                   | TEXT    |
| 25           | Fax                     | TEXT    |
| 26           | AccountNumber           | TEXT    |
| 27           | Website                 | TEXT    |
| 28           | PhotoUrl                | TEXT    |
| 30           | Industry                | TEXT    |
| 31           | AnnualRevenue           | REAL    |
| 32           | NumberOfEmployees       | REAL    |
| 33           | Ownership               | TEXT    |
| 34           | TickerSymbol            | TEXT    |
| 35           | Description             | TEXT    |
| 36           | Rating                  | TEXT    |
| 37           | Site                    | TEXT    |
| 38           | OwnerId                 | TEXT    |
| 39           | CreatedDate             | TEXT    |
| 40           | CreatedById             | TEXT    |
| 41           | LastModifiedDate        | TEXT    |
| 42           | LastModifiedById        | TEXT    |
| 43           | SystemModstamp          | TEXT    |
| 44           | LastActivityDate        | TEXT    |
| 45           | LastViewedDate          | TEXT    |
| 46           | LastReferencedDate      | TEXT    |
| 47           | Jigsaw                  | TEXT    |
| 48           | JigsawCompanyId         | TEXT    |
| 49           | CleanStatus             | TEXT    |
| 50           | AccountSource           | TEXT    |
| 51           | DunsNumber              | TEXT    |
| 52           | Tradestyle              | TEXT    |
| 53           | NaicsCode               | TEXT    |
| 55           | YearStarted             | TEXT    |
| 57           | DandbCompanyId          | TEXT    |
| 58           | OperatingHoursId        | TEXT    |

### salesforce_contact

Represents a contact, which is a person associated with an account.

| Column index | Column name            | type    |
| ------------ | ---------------------- | ------- |
| 0            | Id                     | TEXT    |
| 1            | IsDeleted              | INTEGER |
| 2            | MasterRecordId         | TEXT    |
| 3            | AccountId              | TEXT    |
| 4            | LastName               | TEXT    |
| 5            | FirstName              | TEXT    |
| 6            | Salutation             | TEXT    |
| 7            | Name                   | TEXT    |
| 8            | OtherStreet            | TEXT    |
| 9            | OtherCity              | TEXT    |
| 10           | OtherState             | TEXT    |
| 11           | OtherPostalCode        | TEXT    |
| 12           | OtherCountry           | TEXT    |
| 13           | OtherLatitude          | REAL    |
| 14           | OtherLongitude         | REAL    |
| 15           | OtherGeocodeAccuracy   | TEXT    |
| 16           | OtherAddress           | TEXT    |
| 17           | MailingStreet          | TEXT    |
| 18           | MailingCity            | TEXT    |
| 19           | MailingState           | TEXT    |
| 20           | MailingPostalCode      | TEXT    |
| 21           | MailingCountry         | TEXT    |
| 22           | MailingLatitude        | REAL    |
| 23           | MailingLongitude       | REAL    |
| 24           | MailingGeocodeAccuracy | TEXT    |
| 25           | MailingAddress         | TEXT    |
| 26           | Phone                  | TEXT    |
| 27           | Fax                    | TEXT    |
| 28           | MobilePhone            | TEXT    |
| 29           | HomePhone              | TEXT    |
| 30           | OtherPhone             | TEXT    |
| 31           | AssistantPhone         | TEXT    |
| 32           | ReportsToId            | TEXT    |
| 33           | Email                  | TEXT    |
| 34           | Title                  | TEXT    |
| 35           | Department             | TEXT    |
| 36           | AssistantName          | TEXT    |
| 37           | LeadSource             | TEXT    |
| 38           | Birthdate              | TEXT    |
| 39           | Description            | TEXT    |
| 40           | OwnerId                | TEXT    |
| 41           | CreatedDate            | TEXT    |
| 42           | CreatedById            | TEXT    |
| 43           | LastModifiedDate       | TEXT    |
| 44           | LastModifiedById       | TEXT    |
| 45           | SystemModstamp         | TEXT    |
| 46           | LastActivityDate       | TEXT    |
| 47           | LastCURequestDate      | TEXT    |
| 48           | LastCUUpdateDate       | TEXT    |
| 49           | LastViewedDate         | TEXT    |
| 50           | LastReferencedDate     | TEXT    |
| 51           | EmailBouncedReason     | TEXT    |
| 52           | EmailBouncedDate       | TEXT    |
| 53           | IsEmailBounced         | INTEGER |
| 54           | PhotoUrl               | TEXT    |
| 55           | Jigsaw                 | TEXT    |
| 56           | JigsawContactId        | TEXT    |
| 57           | CleanStatus            | TEXT    |
| 58           | IndividualId           | TEXT    |
| 59           | IsPriorityRecord       | INTEGER |
| 60           | ContactSource          | TEXT    |

### salesforce_lead

Represents a prospect or lead.

| Column index | Column name            | type    |
| ------------ | ---------------------- | ------- |
| 0            | Id                     | TEXT    |
| 1            | IsDeleted              | INTEGER |
| 2            | MasterRecordId         | TEXT    |
| 3            | LastName               | TEXT    |
| 4            | FirstName              | TEXT    |
| 5            | Salutation             | TEXT    |
| 6            | Name                   | TEXT    |
| 7            | Title                  | TEXT    |
| 8            | Company                | TEXT    |
| 9            | Street                 | TEXT    |
| 10           | City                   | TEXT    |
| 11           | State                  | TEXT    |
| 12           | PostalCode             | TEXT    |
| 13           | Country                | TEXT    |
| 14           | Latitude               | REAL    |
| 15           | Longitude              | REAL    |
| 16           | GeocodeAccuracy        | TEXT    |
| 17           | Address                | TEXT    |
| 18           | Phone                  | TEXT    |
| 19           | MobilePhone            | TEXT    |
| 20           | Fax                    | TEXT    |
| 21           | Email                  | TEXT    |
| 22           | Website                | TEXT    |
| 23           | PhotoUrl               | TEXT    |
| 24           | Description            | TEXT    |
| 25           | LeadSource             | TEXT    |
| 26           | Status                 | TEXT    |
| 27           | Industry               | TEXT    |
| 28           | Rating                 | TEXT    |
| 29           | AnnualRevenue          | REAL    |
| 30           | NumberOfEmployees      | REAL    |
| 31           | OwnerId                | TEXT    |
| 32           | IsConverted            | INTEGER |
| 33           | ConvertedDate          | TEXT    |
| 34           | ConvertedAccountId     | TEXT    |
| 35           | ConvertedContactId     | TEXT    |
| 36           | ConvertedOpportunityId | TEXT    |
| 37           | IsUnreadByOwner        | INTEGER |
| 38           | CreatedDate            | TEXT    |
| 39           | CreatedById            | TEXT    |
| 40           | LastModifiedDate       | TEXT    |
| 41           | LastModifiedById       | TEXT    |
| 42           | SystemModstamp         | TEXT    |
| 43           | LastActivityDate       | TEXT    |
| 44           | LastViewedDate         | TEXT    |
| 45           | LastReferencedDate     | TEXT    |
| 46           | Jigsaw                 | TEXT    |
| 47           | JigsawContactId        | TEXT    |
| 48           | CleanStatus            | TEXT    |
| 49           | CompanyDunsNumber      | TEXT    |
| 50           | DandbCompanyId         | TEXT    |
| 51           | EmailBouncedReason     | TEXT    |
| 52           | EmailBouncedDate       | TEXT    |
| 53           | IndividualId           | TEXT    |
| 54           | IsPriorityRecord       | INTEGER |

### salesforce_opportunity

Represents an opportunity, which is a sale or pending deal.

| Column index | Column name                   | type    |
| ------------ | ----------------------------- | ------- |
| 0            | Id                            | TEXT    |
| 1            | IsDeleted                     | INTEGER |
| 2            | AccountId                     | TEXT    |
| 3            | IsPrivate                     | INTEGER |
| 4            | Name                          | TEXT    |
| 5            | Description                   | TEXT    |
| 6            | StageName                     | TEXT    |
| 7            | Amount                        | REAL    |
| 8            | Probability                   | REAL    |
| 9            | ExpectedRevenue               | REAL    |
| 10           | TotalOpportunityQuantity      | REAL    |
| 11           | CloseDate                     | TEXT    |
| 12           | Type                          | TEXT    |
| 13           | NextStep                      | TEXT    |
| 14           | LeadSource                    | TEXT    |
| 15           | IsClosed                      | INTEGER |
| 16           | IsWon                         | INTEGER |
| 17           | ForecastCategory              | TEXT    |
| 18           | ForecastCategoryName          | TEXT    |
| 19           | CampaignId                    | TEXT    |
| 20           | HasOpportunityLineItem        | INTEGER |
| 21           | Pricebook2Id                  | TEXT    |
| 22           | OwnerId                       | TEXT    |
| 23           | CreatedDate                   | TEXT    |
| 24           | CreatedById                   | TEXT    |
| 25           | LastModifiedDate              | TEXT    |
| 26           | LastModifiedById              | TEXT    |
| 27           | SystemModstamp                | TEXT    |
| 28           | LastActivityDate              | TEXT    |
| 29           | PushCount                     | REAL    |
| 30           | LastStageChangeDate           | TEXT    |
| 31           | FiscalQuarter                 | REAL    |
| 32           | FiscalYear                    | REAL    |
| 33           | Fiscal                        | TEXT    |
| 34           | ContactId                     | TEXT    |
| 35           | LastViewedDate                | TEXT    |
| 36           | LastReferencedDate            | TEXT    |
| 37           | SyncedQuoteId                 | TEXT    |
| 38           | HasOpenActivity               | INTEGER |
| 39           | HasOverdueTask                | INTEGER |
| 40           | LastAmountChangedHistoryId    | TEXT    |
| 41           | LastCloseDateChangedHistoryId | TEXT    |

### salesforce_case

Represents a case, which is a customer issue or problem.

| Column index | Column name        | type    |
| ------------ | ------------------ | ------- |
| 0            | Id                 | TEXT    |
| 1            | IsDeleted          | INTEGER |
| 2            | MasterRecordId     | TEXT    |
| 3            | CaseNumber         | TEXT    |
| 4            | ContactId          | TEXT    |
| 5            | AccountId          | TEXT    |
| 6            | AssetId            | TEXT    |
| 7            | SourceId           | TEXT    |
| 8            | ParentId           | TEXT    |
| 9            | SuppliedName       | TEXT    |
| 10           | SuppliedEmail      | TEXT    |
| 11           | SuppliedPhone      | TEXT    |
| 12           | SuppliedCompany    | TEXT    |
| 13           | Type               | TEXT    |
| 14           | Status             | TEXT    |
| 15           | Reason             | TEXT    |
| 16           | Origin             | TEXT    |
| 17           | Subject            | TEXT    |
| 18           | Priority           | TEXT    |
| 19           | Description        | TEXT    |
| 20           | IsClosed           | INTEGER |
| 21           | ClosedDate         | TEXT    |
| 22           | IsEscalated        | INTEGER |
| 23           | OwnerId            | TEXT    |
| 24           | CreatedDate        | TEXT    |
| 25           | CreatedById        | TEXT    |
| 26           | LastModifiedDate   | TEXT    |
| 27           | LastModifiedById   | TEXT    |
| 28           | SystemModstamp     | TEXT    |
| 29           | ContactPhone       | TEXT    |
| 30           | ContactMobile      | TEXT    |
| 31           | ContactEmail       | TEXT    |
| 32           | ContactFax         | TEXT    |
| 33           | Comments           | TEXT    |
| 34           | LastViewedDate     | TEXT    |
| 35           | LastReferencedDate | TEXT    |

### salesforce_task

Represents a business activity such as making a phone call or other to-do items. In the user interface, Task and Event records are collectively referred to as activities.

| Column index | Column name               | type    |
| ------------ | ------------------------- | ------- |
| 0            | Id                        | TEXT    |
| 1            | WhoId                     | TEXT    |
| 2            | WhatId                    | TEXT    |
| 3            | Subject                   | TEXT    |
| 4            | ActivityDate              | TEXT    |
| 5            | Status                    | TEXT    |
| 6            | Priority                  | TEXT    |
| 7            | IsHighPriority            | INTEGER |
| 8            | OwnerId                   | TEXT    |
| 9            | Description               | TEXT    |
| 10           | IsDeleted                 | INTEGER |
| 11           | AccountId                 | TEXT    |
| 12           | IsClosed                  | INTEGER |
| 13           | CreatedDate               | TEXT    |
| 14           | CreatedById               | TEXT    |
| 15           | LastModifiedDate          | TEXT    |
| 16           | LastModifiedById          | TEXT    |
| 17           | SystemModstamp            | TEXT    |
| 18           | IsArchived                | INTEGER |
| 19           | CallDurationInSeconds     | REAL    |
| 20           | CallType                  | TEXT    |
| 21           | CallDisposition           | TEXT    |
| 22           | CallObject                | TEXT    |
| 23           | ReminderDateTime          | TEXT    |
| 24           | IsReminderSet             | INTEGER |
| 25           | RecurrenceActivityId      | TEXT    |
| 26           | IsRecurrence              | INTEGER |
| 27           | RecurrenceStartDateOnly   | TEXT    |
| 28           | RecurrenceEndDateOnly     | TEXT    |
| 29           | RecurrenceTimeZoneSidKey  | TEXT    |
| 30           | RecurrenceType            | TEXT    |
| 31           | RecurrenceInterval        | REAL    |
| 32           | RecurrenceDayOfWeekMask   | REAL    |
| 33           | RecurrenceDayOfMonth      | REAL    |
| 34           | RecurrenceInstance        | TEXT    |
| 35           | RecurrenceMonthOfYear     | TEXT    |
| 36           | RecurrenceRegeneratedType | TEXT    |
| 37           | TaskSubtype               | TEXT    |
| 38           | CompletedDateTime         | TEXT    |

### salesforce_event

Represents an event in the calendar. In the user interface, event and task records are collectively referred to as activities.

| Column index | Column name                 | type    |
| ------------ | --------------------------- | ------- |
| 0            | Id                          | TEXT    |
| 1            | WhoId                       | TEXT    |
| 2            | WhatId                      | TEXT    |
| 3            | Subject                     | TEXT    |
| 4            | Location                    | TEXT    |
| 5            | IsAllDayEvent               | INTEGER |
| 6            | ActivityDateTime            | TEXT    |
| 7            | ActivityDate                | TEXT    |
| 8            | DurationInMinutes           | REAL    |
| 9            | StartDateTime               | TEXT    |
| 10           | EndDateTime                 | TEXT    |
| 11           | EndDate                     | TEXT    |
| 12           | Description                 | TEXT    |
| 13           | AccountId                   | TEXT    |
| 14           | OwnerId                     | TEXT    |
| 15           | IsPrivate                   | INTEGER |
| 16           | ShowAs                      | TEXT    |
| 17           | IsDeleted                   | INTEGER |
| 18           | IsChild                     | INTEGER |
| 19           | IsGroupEvent                | INTEGER |
| 20           | GroupEventType              | TEXT    |
| 21           | CreatedDate                 | TEXT    |
| 22           | CreatedById                 | TEXT    |
| 23           | LastModifiedDate            | TEXT    |
| 24           | LastModifiedById            | TEXT    |
| 25           | SystemModstamp              | TEXT    |
| 26           | IsArchived                  | INTEGER |
| 27           | RecurrenceActivityId        | TEXT    |
| 28           | IsRecurrence                | INTEGER |
| 29           | RecurrenceStartDateTime     | TEXT    |
| 30           | RecurrenceEndDateOnly       | TEXT    |
| 31           | RecurrenceTimeZoneSidKey    | TEXT    |
| 32           | RecurrenceType              | TEXT    |
| 33           | RecurrenceInterval          | REAL    |
| 34           | RecurrenceDayOfWeekMask     | REAL    |
| 35           | RecurrenceDayOfMonth        | REAL    |
| 36           | RecurrenceInstance          | TEXT    |
| 37           | RecurrenceMonthOfYear       | TEXT    |
| 38           | ReminderDateTime            | TEXT    |
| 39           | IsReminderSet               | INTEGER |
| 40           | EventSubtype                | TEXT    |
| 41           | IsRecurrence2Exclusion      | INTEGER |
| 42           | Recurrence2PatternText      | TEXT    |
| 43           | Recurrence2PatternVersion   | TEXT    |
| 44           | IsRecurrence2               | INTEGER |
| 45           | IsRecurrence2Exception      | INTEGER |
| 46           | Recurrence2PatternStartDate | TEXT    |
| 47           | Recurrence2PatternTimeZone  | TEXT    |
| 48           | ServiceAppointmentId        | TEXT    |

### salesforce_campaign

Represents and tracks a marketing campaign, such as a direct mail promotion, webinar, or trade show.

| Column index | Column name                | type    |
| ------------ | -------------------------- | ------- |
| 0            | Id                         | TEXT    |
| 1            | IsDeleted                  | INTEGER |
| 2            | Name                       | TEXT    |
| 3            | ParentId                   | TEXT    |
| 4            | Type                       | TEXT    |
| 5            | Status                     | TEXT    |
| 6            | StartDate                  | TEXT    |
| 7            | EndDate                    | TEXT    |
| 8            | ExpectedRevenue            | REAL    |
| 9            | BudgetedCost               | REAL    |
| 10           | ActualCost                 | REAL    |
| 11           | ExpectedResponse           | REAL    |
| 12           | NumberSent                 | REAL    |
| 13           | IsActive                   | INTEGER |
| 14           | Description                | TEXT    |
| 15           | NumberOfLeads              | REAL    |
| 16           | NumberOfConvertedLeads     | REAL    |
| 17           | NumberOfContacts           | REAL    |
| 18           | NumberOfResponses          | REAL    |
| 19           | NumberOfOpportunities      | REAL    |
| 20           | NumberOfWonOpportunities   | REAL    |
| 21           | AmountAllOpportunities     | REAL    |
| 22           | AmountWonOpportunities     | REAL    |
| 23           | OwnerId                    | TEXT    |
| 24           | CreatedDate                | TEXT    |
| 25           | CreatedById                | TEXT    |
| 26           | LastModifiedDate           | TEXT    |
| 27           | LastModifiedById           | TEXT    |
| 28           | SystemModstamp             | TEXT    |
| 29           | LastActivityDate           | TEXT    |
| 30           | LastViewedDate             | TEXT    |
| 31           | LastReferencedDate         | TEXT    |
| 32           | CampaignMemberRecordTypeId | TEXT    |

### salesforce_user

Represents a user in your organization.

| Column index | Column name                                            | type    |
| ------------ | ------------------------------------------------------ | ------- |
| 0            | Id                                                     | TEXT    |
| 1            | Username                                               | TEXT    |
| 2            | LastName                                               | TEXT    |
| 3            | FirstName                                              | TEXT    |
| 4            | Name                                                   | TEXT    |
| 5            | CompanyName                                            | TEXT    |
| 6            | Division                                               | TEXT    |
| 7            | Department                                             | TEXT    |
| 8            | Title                                                  | TEXT    |
| 9            | Street                                                 | TEXT    |
| 10           | City                                                   | TEXT    |
| 11           | State                                                  | TEXT    |
| 12           | PostalCode                                             | TEXT    |
| 13           | Country                                                | TEXT    |
| 14           | Latitude                                               | REAL    |
| 15           | Longitude                                              | REAL    |
| 16           | GeocodeAccuracy                                        | TEXT    |
| 17           | Address                                                | TEXT    |
| 18           | Email                                                  | TEXT    |
| 20           | EmailPreferencesAutoBccStayInTouch                     | INTEGER |
| 21           | EmailPreferencesStayInTouchReminder                    | INTEGER |
| 22           | SenderEmail                                            | TEXT    |
| 23           | SenderName                                             | TEXT    |
| 24           | Signature                                              | TEXT    |
| 25           | StayInTouchSubject                                     | TEXT    |
| 26           | StayInTouchSignature                                   | TEXT    |
| 27           | StayInTouchNote                                        | TEXT    |
| 28           | Phone                                                  | TEXT    |
| 29           | Fax                                                    | TEXT    |
| 30           | MobilePhone                                            | TEXT    |
| 31           | Alias                                                  | TEXT    |
| 32           | CommunityNickname                                      | TEXT    |
| 33           | BadgeText                                              | TEXT    |
| 34           | IsActive                                               | INTEGER |
| 35           | TimeZoneSidKey                                         | TEXT    |
| 36           | UserRoleId                                             | TEXT    |
| 37           | LocaleSidKey                                           | TEXT    |
| 38           | ReceivesInfoEmails                                     | INTEGER |
| 39           | ReceivesAdminInfoEmails                                | INTEGER |
| 40           | EmailEncodingKey                                       | TEXT    |
| 41           | ProfileId                                              | TEXT    |
| 42           | UserType                                               | TEXT    |
| 43           | LanguageLocaleKey                                      | TEXT    |
| 44           | EmployeeNumber                                         | TEXT    |
| 45           | DelegatedApproverId                                    | TEXT    |
| 46           | ManagerId                                              | TEXT    |
| 47           | LastLoginDate                                          | TEXT    |
| 48           | LastPasswordChangeDate                                 | TEXT    |
| 49           | CreatedDate                                            | TEXT    |
| 50           | CreatedById                                            | TEXT    |
| 51           | LastModifiedDate                                       | TEXT    |
| 52           | LastModifiedById                                       | TEXT    |
| 53           | SystemModstamp                                         | TEXT    |
| 54           | NumberOfFailedLogins                                   | REAL    |
| 55           | OfflineTrialExpirationDate                             | TEXT    |
| 56           | OfflinePdaTrialExpirationDate                          | TEXT    |
| 57           | UserPermissionsMarketingUser                           | INTEGER |
| 58           | UserPermissionsOfflineUser                             | INTEGER |
| 59           | UserPermissionsCallCenterAutoLogin                     | INTEGER |
| 60           | UserPermissionsSFContentUser                           | INTEGER |
| 61           | UserPermissionsKnowledgeUser                           | INTEGER |
| 62           | UserPermissionsInteractionUser                         | INTEGER |
| 63           | UserPermissionsSupportUser                             | INTEGER |
| 64           | UserPermissionsJigsawProspectingUser                   | INTEGER |
| 65           | UserPermissionsSiteforceContributorUser                | INTEGER |
| 66           | UserPermissionsSiteforcePublisherUser                  | INTEGER |
| 67           | UserPermissionsWorkDotComUserFeature                   | INTEGER |
| 68           | ForecastEnabled                                        | INTEGER |
| 69           | UserPreferencesActivityRemindersPopup                  | INTEGER |
| 70           | UserPreferencesEventRemindersCheckboxDefault           | INTEGER |
| 71           | UserPreferencesTaskRemindersCheckboxDefault            | INTEGER |
| 72           | UserPreferencesReminderSoundOff                        | INTEGER |
| 73           | UserPreferencesDisableAllFeedsEmail                    | INTEGER |
| 74           | UserPreferencesDisableFollowersEmail                   | INTEGER |
| 75           | UserPreferencesDisableProfilePostEmail                 | INTEGER |
| 76           | UserPreferencesDisableChangeCommentEmail               | INTEGER |
| 77           | UserPreferencesDisableLaterCommentEmail                | INTEGER |
| 78           | UserPreferencesDisProfPostCommentEmail                 | INTEGER |
| 79           | UserPreferencesContentNoEmail                          | INTEGER |
| 80           | UserPreferencesContentEmailAsAndWhen                   | INTEGER |
| 81           | UserPreferencesApexPagesDeveloperMode                  | INTEGER |
| 82           | UserPreferencesReceiveNoNotificationsAsApprover        | INTEGER |
| 83           | UserPreferencesReceiveNotificationsAsDelegatedApprover | INTEGER |
| 84           | UserPreferencesHideCSNGetChatterMobileTask             | INTEGER |
| 85           | UserPreferencesDisableMentionsPostEmail                | INTEGER |
| 86           | UserPreferencesDisMentionsCommentEmail                 | INTEGER |
| 87           | UserPreferencesHideCSNDesktopTask                      | INTEGER |
| 88           | UserPreferencesHideChatterOnboardingSplash             | INTEGER |
| 89           | UserPreferencesHideSecondChatterOnboardingSplash       | INTEGER |
| 90           | UserPreferencesDisCommentAfterLikeEmail                | INTEGER |
| 91           | UserPreferencesDisableLikeEmail                        | INTEGER |
| 92           | UserPreferencesSortFeedByComment                       | INTEGER |
| 93           | UserPreferencesDisableMessageEmail                     | INTEGER |
| 94           | UserPreferencesJigsawListUser                          | INTEGER |
| 95           | UserPreferencesDisableBookmarkEmail                    | INTEGER |
| 96           | UserPreferencesDisableSharePostEmail                   | INTEGER |
| 97           | UserPreferencesEnableAutoSubForFeeds                   | INTEGER |
| 98           | UserPreferencesDisableFileShareNotificationsForApi     | INTEGER |
| 99           | UserPreferencesShowTitleToExternalUsers                | INTEGER |
| 100          | UserPreferencesShowManagerToExternalUsers              | INTEGER |
| 101          | UserPreferencesShowEmailToExternalUsers                | INTEGER |
| 102          | UserPreferencesShowWorkPhoneToExternalUsers            | INTEGER |
| 103          | UserPreferencesShowMobilePhoneToExternalUsers          | INTEGER |
| 104          | UserPreferencesShowFaxToExternalUsers                  | INTEGER |
| 105          | UserPreferencesShowStreetAddressToExternalUsers        | INTEGER |
| 106          | UserPreferencesShowCityToExternalUsers                 | INTEGER |
| 107          | UserPreferencesShowStateToExternalUsers                | INTEGER |
| 108          | UserPreferencesShowPostalCodeToExternalUsers           | INTEGER |
| 109          | UserPreferencesShowCountryToExternalUsers              | INTEGER |
| 110          | UserPreferencesShowProfilePicToGuestUsers              | INTEGER |
| 111          | UserPreferencesShowTitleToGuestUsers                   | INTEGER |
| 112          | UserPreferencesShowCityToGuestUsers                    | INTEGER |
| 113          | UserPreferencesShowStateToGuestUsers                   | INTEGER |
| 114          | UserPreferencesShowPostalCodeToGuestUsers              | INTEGER |
| 115          | UserPreferencesShowCountryToGuestUsers                 | INTEGER |
| 116          | UserPreferencesShowForecastingChangeSignals            | INTEGER |
| 117          | UserPreferencesLiveAgentMiawSetupDeflection            | INTEGER |
| 118          | UserPreferencesHideS1BrowserUI                         | INTEGER |
| 119          | UserPreferencesDisableEndorsementEmail                 | INTEGER |
| 120          | UserPreferencesPathAssistantCollapsed                  | INTEGER |
| 121          | UserPreferencesCacheDiagnostics                        | INTEGER |
| 122          | UserPreferencesShowEmailToGuestUsers                   | INTEGER |
| 123          | UserPreferencesShowManagerToGuestUsers                 | INTEGER |
| 124          | UserPreferencesShowWorkPhoneToGuestUsers               | INTEGER |
| 125          | UserPreferencesShowMobilePhoneToGuestUsers             | INTEGER |
| 126          | UserPreferencesShowFaxToGuestUsers                     | INTEGER |
| 127          | UserPreferencesShowStreetAddressToGuestUsers           | INTEGER |
| 128          | UserPreferencesLightningExperiencePreferred            | INTEGER |
| 129          | UserPreferencesPreviewLightning                        | INTEGER |
| 130          | UserPreferencesHideEndUserOnboardingAssistantModal     | INTEGER |
| 131          | UserPreferencesHideLightningMigrationModal             | INTEGER |
| 132          | UserPreferencesHideSfxWelcomeMat                       | INTEGER |
| 133          | UserPreferencesHideBiggerPhotoCallout                  | INTEGER |
| 134          | UserPreferencesGlobalNavBarWTShown                     | INTEGER |
| 135          | UserPreferencesGlobalNavGridMenuWTShown                | INTEGER |
| 136          | UserPreferencesCreateLEXAppsWTShown                    | INTEGER |
| 137          | UserPreferencesFavoritesWTShown                        | INTEGER |
| 138          | UserPreferencesRecordHomeSectionCollapseWTShown        | INTEGER |
| 139          | UserPreferencesRecordHomeReservedWTShown               | INTEGER |
| 140          | UserPreferencesFavoritesShowTopFavorites               | INTEGER |
| 141          | UserPreferencesExcludeMailAppAttachments               | INTEGER |
| 142          | UserPreferencesSuppressTaskSFXReminders                | INTEGER |
| 143          | UserPreferencesSuppressEventSFXReminders               | INTEGER |
| 144          | UserPreferencesPreviewCustomTheme                      | INTEGER |
| 145          | UserPreferencesHasCelebrationBadge                     | INTEGER |
| 146          | UserPreferencesUserDebugModePref                       | INTEGER |
| 147          | UserPreferencesSRHOverrideActivities                   | INTEGER |
| 148          | UserPreferencesNewLightningReportRunPageEnabled        | INTEGER |
| 149          | UserPreferencesReverseOpenActivitiesView               | INTEGER |
| 150          | UserPreferencesShowTerritoryTimeZoneShifts             | INTEGER |
| 151          | UserPreferencesHasSentWarningEmail                     | INTEGER |
| 152          | UserPreferencesHasSentWarningEmail238                  | INTEGER |
| 153          | UserPreferencesHasSentWarningEmail240                  | INTEGER |
| 154          | UserPreferencesNativeEmailClient                       | INTEGER |
| 155          | UserPreferencesShowForecastingRoundedAmounts           | INTEGER |
| 156          | ContactId                                              | TEXT    |
| 157          | AccountId                                              | TEXT    |
| 158          | CallCenterId                                           | TEXT    |
| 159          | Extension                                              | TEXT    |
| 160          | FederationIdentifier                                   | TEXT    |
| 161          | AboutMe                                                | TEXT    |
| 162          | FullPhotoUrl                                           | TEXT    |
| 163          | SmallPhotoUrl                                          | TEXT    |
| 164          | IsExtIndicatorVisible                                  | INTEGER |
| 165          | OutOfOfficeMessage                                     | TEXT    |
| 166          | MediumPhotoUrl                                         | TEXT    |
| 167          | DigestFrequency                                        | TEXT    |
| 168          | DefaultGroupNotificationFrequency                      | TEXT    |
| 169          | JigsawImportLimitOverride                              | REAL    |
| 170          | LastViewedDate                                         | TEXT    |
| 171          | LastReferencedDate                                     | TEXT    |
| 172          | BannerPhotoUrl                                         | TEXT    |
| 173          | SmallBannerPhotoUrl                                    | TEXT    |
| 174          | MediumBannerPhotoUrl                                   | TEXT    |
| 175          | IsProfilePhotoActive                                   | INTEGER |
| 176          | IndividualId                                           | TEXT    |

### salesforce_campaignmember

The CampaignMember object represents the relationship between a campaign and either a lead or a contact. If the Accounts as Campaign Members setting is enabled in an org, CampaignMember can also represent the relationship between a campaign and an account.

| Column index | Column name          | type    |
| ------------ | -------------------- | ------- |
| 0            | Id                   | TEXT    |
| 1            | IsDeleted            | INTEGER |
| 2            | CampaignId           | TEXT    |
| 3            | LeadId               | TEXT    |
| 4            | ContactId            | TEXT    |
| 5            | Status               | TEXT    |
| 6            | HasResponded         | INTEGER |
| 7            | CreatedDate          | TEXT    |
| 8            | CreatedById          | TEXT    |
| 9            | LastModifiedDate     | TEXT    |
| 10           | LastModifiedById     | TEXT    |
| 11           | SystemModstamp       | TEXT    |
| 12           | FirstRespondedDate   | TEXT    |
| 13           | Salutation           | TEXT    |
| 14           | Name                 | TEXT    |
| 15           | FirstName            | TEXT    |
| 16           | LastName             | TEXT    |
| 17           | Title                | TEXT    |
| 18           | Street               | TEXT    |
| 19           | City                 | TEXT    |
| 20           | State                | TEXT    |
| 21           | PostalCode           | TEXT    |
| 22           | Country              | TEXT    |
| 23           | Email                | TEXT    |
| 24           | Phone                | TEXT    |
| 25           | Fax                  | TEXT    |
| 26           | MobilePhone          | TEXT    |
| 27           | Description          | TEXT    |
| 28           | DoNotCall            | INTEGER |
| 29           | HasOptedOutOfEmail   | INTEGER |
| 30           | HasOptedOutOfFax     | INTEGER |
| 31           | LeadSource           | TEXT    |
| 32           | CompanyOrAccount     | TEXT    |
| 33           | Type                 | TEXT    |
| 34           | LeadOrContactId      | TEXT    |
| 35           | LeadOrContactOwnerId | TEXT    |

### salesforce_asset

Represents an item of commercial value, such as a product sold by your company or a competitor, that a customer has purchased.

| Column index | Column name             | type    |
| ------------ | ----------------------- | ------- |
| 0            | Id                      | TEXT    |
| 1            | ContactId               | TEXT    |
| 2            | AccountId               | TEXT    |
| 3            | ParentId                | TEXT    |
| 4            | RootAssetId             | TEXT    |
| 5            | Product2Id              | TEXT    |
| 6            | ProductCode             | TEXT    |
| 7            | IsCompetitorProduct     | INTEGER |
| 8            | CreatedDate             | TEXT    |
| 9            | CreatedById             | TEXT    |
| 10           | LastModifiedDate        | TEXT    |
| 11           | LastModifiedById        | TEXT    |
| 12           | SystemModstamp          | TEXT    |
| 13           | IsDeleted               | INTEGER |
| 14           | Name                    | TEXT    |
| 15           | SerialNumber            | TEXT    |
| 16           | InstallDate             | TEXT    |
| 17           | PurchaseDate            | TEXT    |
| 18           | UsageEndDate            | TEXT    |
| 19           | LifecycleStartDate      | TEXT    |
| 20           | LifecycleEndDate        | TEXT    |
| 21           | Status                  | TEXT    |
| 22           | Price                   | REAL    |
| 23           | Quantity                | REAL    |
| 24           | Description             | TEXT    |
| 25           | OwnerId                 | TEXT    |
| 26           | AssetProvidedById       | TEXT    |
| 27           | AssetServicedById       | TEXT    |
| 28           | IsInternal              | INTEGER |
| 29           | AssetLevel              | REAL    |
| 30           | StockKeepingUnit        | TEXT    |
| 31           | HasLifecycleManagement  | INTEGER |
| 32           | CurrentMrr              | REAL    |
| 33           | CurrentLifecycleEndDate | TEXT    |
| 34           | CurrentQuantity         | REAL    |
| 35           | CurrentAmount           | REAL    |
| 36           | TotalLifecycleAmount    | REAL    |
| 37           | Street                  | TEXT    |
| 38           | City                    | TEXT    |
| 39           | State                   | TEXT    |
| 40           | PostalCode              | TEXT    |
| 41           | Country                 | TEXT    |
| 42           | Latitude                | REAL    |
| 43           | Longitude               | REAL    |
| 44           | GeocodeAccuracy         | TEXT    |
| 45           | Address                 | TEXT    |
| 46           | LastViewedDate          | TEXT    |
| 47           | LastReferencedDate      | TEXT    |

### salesforce_contract

Represents a contract (a business agreement) associated with an Account.

| Column index | Column name            | type    |
| ------------ | ---------------------- | ------- |
| 0            | Id                     | TEXT    |
| 1            | AccountId              | TEXT    |
| 2            | Pricebook2Id           | TEXT    |
| 3            | OwnerExpirationNotice  | TEXT    |
| 4            | StartDate              | TEXT    |
| 5            | EndDate                | TEXT    |
| 6            | BillingStreet          | TEXT    |
| 7            | BillingCity            | TEXT    |
| 8            | BillingState           | TEXT    |
| 9            | BillingPostalCode      | TEXT    |
| 10           | BillingCountry         | TEXT    |
| 11           | BillingLatitude        | REAL    |
| 12           | BillingLongitude       | REAL    |
| 13           | BillingGeocodeAccuracy | TEXT    |
| 14           | BillingAddress         | TEXT    |
| 15           | ContractTerm           | REAL    |
| 16           | OwnerId                | TEXT    |
| 17           | Status                 | TEXT    |
| 18           | CompanySignedId        | TEXT    |
| 19           | CompanySignedDate      | TEXT    |
| 20           | CustomerSignedId       | TEXT    |
| 21           | CustomerSignedTitle    | TEXT    |
| 22           | CustomerSignedDate     | TEXT    |
| 23           | SpecialTerms           | TEXT    |
| 24           | ActivatedById          | TEXT    |
| 25           | ActivatedDate          | TEXT    |
| 26           | StatusCode             | TEXT    |
| 27           | Description            | TEXT    |
| 28           | IsDeleted              | INTEGER |
| 29           | ContractNumber         | TEXT    |
| 30           | LastApprovedDate       | TEXT    |
| 31           | CreatedDate            | TEXT    |
| 32           | CreatedById            | TEXT    |
| 33           | LastModifiedDate       | TEXT    |
| 34           | LastModifiedById       | TEXT    |
| 35           | SystemModstamp         | TEXT    |
| 36           | LastActivityDate       | TEXT    |
| 37           | LastViewedDate         | TEXT    |
| 38           | LastReferencedDate     | TEXT    |

### salesforce_contractlineitem

Represents a product covered by a service contract (customer support agreement).

| Column index | Column name              | type    |
| ------------ | ------------------------ | ------- |
| 0            | Id                       | TEXT    |
| 1            | IsDeleted                | INTEGER |
| 2            | LineItemNumber           | TEXT    |
| 3            | CreatedDate              | TEXT    |
| 4            | CreatedById              | TEXT    |
| 5            | LastModifiedDate         | TEXT    |
| 6            | LastModifiedById         | TEXT    |
| 7            | SystemModstamp           | TEXT    |
| 8            | LastViewedDate           | TEXT    |
| 9            | LastReferencedDate       | TEXT    |
| 10           | ServiceContractId        | TEXT    |
| 11           | Product2Id               | TEXT    |
| 12           | AssetId                  | TEXT    |
| 13           | StartDate                | TEXT    |
| 14           | EndDate                  | TEXT    |
| 15           | Description              | TEXT    |
| 16           | PricebookEntryId         | TEXT    |
| 17           | Quantity                 | REAL    |
| 18           | UnitPrice                | REAL    |
| 19           | Discount                 | REAL    |
| 20           | ListPrice                | REAL    |
| 21           | Subtotal                 | REAL    |
| 22           | TotalPrice               | REAL    |
| 23           | Status                   | TEXT    |
| 24           | ParentContractLineItemId | TEXT    |
| 25           | RootContractLineItemId   | TEXT    |
| 26           | LocationId               | TEXT    |

### salesforce_servicecontract

Represents a customer support contract (business agreement).

| Column index | Column name             | type    |
| ------------ | ----------------------- | ------- |
| 0            | Id                      | TEXT    |
| 1            | OwnerId                 | TEXT    |
| 2            | IsDeleted               | INTEGER |
| 3            | Name                    | TEXT    |
| 4            | CreatedDate             | TEXT    |
| 5            | CreatedById             | TEXT    |
| 6            | LastModifiedDate        | TEXT    |
| 7            | LastModifiedById        | TEXT    |
| 8            | SystemModstamp          | TEXT    |
| 9            | LastViewedDate          | TEXT    |
| 10           | LastReferencedDate      | TEXT    |
| 11           | AccountId               | TEXT    |
| 12           | ContactId               | TEXT    |
| 13           | Term                    | REAL    |
| 14           | StartDate               | TEXT    |
| 15           | EndDate                 | TEXT    |
| 16           | ActivationDate          | TEXT    |
| 17           | ApprovalStatus          | TEXT    |
| 18           | Description             | TEXT    |
| 19           | BillingStreet           | TEXT    |
| 20           | BillingCity             | TEXT    |
| 21           | BillingState            | TEXT    |
| 22           | BillingPostalCode       | TEXT    |
| 23           | BillingCountry          | TEXT    |
| 24           | BillingLatitude         | REAL    |
| 25           | BillingLongitude        | REAL    |
| 26           | BillingGeocodeAccuracy  | TEXT    |
| 27           | BillingAddress          | TEXT    |
| 28           | ShippingStreet          | TEXT    |
| 29           | ShippingCity            | TEXT    |
| 30           | ShippingState           | TEXT    |
| 31           | ShippingPostalCode      | TEXT    |
| 32           | ShippingCountry         | TEXT    |
| 33           | ShippingLatitude        | REAL    |
| 34           | ShippingLongitude       | REAL    |
| 35           | ShippingGeocodeAccuracy | TEXT    |
| 36           | ShippingAddress         | TEXT    |
| 37           | Pricebook2Id            | TEXT    |
| 38           | ShippingHandling        | REAL    |
| 39           | Tax                     | REAL    |
| 40           | Subtotal                | REAL    |
| 41           | TotalPrice              | REAL    |
| 42           | LineItemCount           | REAL    |
| 43           | ContractNumber          | TEXT    |
| 44           | SpecialTerms            | TEXT    |
| 45           | Discount                | REAL    |
| 46           | GrandTotal              | REAL    |
| 47           | Status                  | TEXT    |
| 48           | ParentServiceContractId | TEXT    |
| 49           | RootServiceContractId   | TEXT    |
| 50           | AdditionalDiscount      | REAL    |

### salesforce_solution

Represents a detailed description of a customer issue and the resolution of that issue.

| Column index | Column name           | type    |
| ------------ | --------------------- | ------- |
| 0            | Id                    | TEXT    |
| 1            | IsDeleted             | INTEGER |
| 2            | SolutionNumber        | TEXT    |
| 3            | SolutionName          | TEXT    |
| 4            | IsPublished           | INTEGER |
| 5            | IsPublishedInPublicKb | INTEGER |
| 6            | Status                | TEXT    |
| 7            | IsReviewed            | INTEGER |
| 8            | SolutionNote          | TEXT    |
| 9            | OwnerId               | TEXT    |
| 10           | CreatedDate           | TEXT    |
| 11           | CreatedById           | TEXT    |
| 12           | LastModifiedDate      | TEXT    |
| 13           | LastModifiedById      | TEXT    |
| 14           | SystemModstamp        | TEXT    |
| 15           | TimesUsed             | REAL    |
| 16           | LastViewedDate        | TEXT    |
| 17           | LastReferencedDate    | TEXT    |
| 18           | IsHtml                | INTEGER |

### salesforce_pricebook2

Represents a price book that contains the list of products that your org sells.

| Column index | Column name        | type    |
| ------------ | ------------------ | ------- |
| 0            | Id                 | TEXT    |
| 1            | IsDeleted          | INTEGER |
| 2            | Name               | TEXT    |
| 3            | CreatedDate        | TEXT    |
| 4            | CreatedById        | TEXT    |
| 5            | LastModifiedDate   | TEXT    |
| 6            | LastModifiedById   | TEXT    |
| 7            | SystemModstamp     | TEXT    |
| 8            | LastViewedDate     | TEXT    |
| 9            | LastReferencedDate | TEXT    |
| 10           | IsActive           | INTEGER |
| 11           | IsArchived         | INTEGER |
| 12           | Description        | TEXT    |
| 13           | IsStandard         | INTEGER |

### salesforce_product2

Represents a product that your company sells.
This table has several fields that are used only for quantity and revenue schedules (for example, annuities). Schedules are available only for orgs that have enabled the products and schedules features. If these features aren’t enabled, the schedule fields don’t appear , and you can’t query, create, or update the fields.

Use this table to define the default product information for your org. This table is associated by reference with Pricebook2 table via PricebookEntry objects. The same product can be represented in different price books as price book entries. In fact, the same product can be represented multiple times (as separate PricebookEntry records) in the same price book with different prices or currencies. A product can only have one price for a given currency within the same price book. To be used in custom price books, all standard prices must be added as price book entries to the standard price book.

| Column index | Column name           | type    |
| ------------ | --------------------- | ------- |
| 0            | Id                    | TEXT    |
| 1            | Name                  | TEXT    |
| 2            | ProductCode           | TEXT    |
| 3            | Description           | TEXT    |
| 4            | IsActive              | INTEGER |
| 5            | CreatedDate           | TEXT    |
| 6            | CreatedById           | TEXT    |
| 7            | LastModifiedDate      | TEXT    |
| 8            | LastModifiedById      | TEXT    |
| 9            | SystemModstamp        | TEXT    |
| 10           | Family                | TEXT    |
| 11           | ExternalDataSourceId  | TEXT    |
| 12           | ExternalId            | TEXT    |
| 13           | DisplayUrl            | TEXT    |
| 14           | QuantityUnitOfMeasure | TEXT    |
| 15           | IsDeleted             | INTEGER |
| 16           | IsArchived            | INTEGER |
| 17           | LastViewedDate        | TEXT    |
| 18           | LastReferencedDate    | TEXT    |
| 19           | StockKeepingUnit      | TEXT    |
| 20           | Type                  | TEXT    |
| 21           | ProductClass          | TEXT    |

### salesforce_productitem

Represents the stock of a particular product at a particular location in field service, such as all bolts stored in your main warehouse. Each product item is associated with a product and a location in Salesforce. If a product is stored at multiple locations, the product will be tracked in a different product item for each location.

Note that [field service](https://help.salesforce.com/s/articleView?id=sf.fs_overview.htm&type=5) must be enabled in your org to use this table.

| Column index | Column name           | type    |
| ------------ | --------------------- | ------- |
| 0            | Id                    | TEXT    |
| 1            | OwnerId               | TEXT    |
| 2            | IsDeleted             | INTEGER |
| 3            | ProductItemNumber     | TEXT    |
| 4            | CreatedDate           | TEXT    |
| 5            | CreatedById           | TEXT    |
| 6            | LastModifiedDate      | TEXT    |
| 7            | LastModifiedById      | TEXT    |
| 8            | SystemModstamp        | TEXT    |
| 9            | LastViewedDate        | TEXT    |
| 10           | LastReferencedDate    | TEXT    |
| 11           | LocationId            | TEXT    |
| 12           | Product2Id            | TEXT    |
| 13           | ProductName           | TEXT    |
| 14           | SerialNumber          | TEXT    |
| 15           | QuantityOnHand        | REAL    |
| 16           | QuantityUnitOfMeasure | TEXT    |
| 17           | IsProduct2Serialized  | INTEGER |

### salesforce_pricebookentry

Represents a product entry (an association between a Pricebook2 and Product2) in a price book.

Use this table to define the association between your organization’s products (Product2) and your organization’s standard price book or to custom price books ( Pricebook2). Create one PricebookEntry record for each standard or custom price and currency combination for a product in a Pricebook2.

When creating these records, you must specify the IDs of the associated Pricebook2 record and Product2 record. Once these records are created, you can’t update these IDs.

This table is defined only for those organizations that have products enabled as a feature.

| Column index | Column name      | type    |
| ------------ | ---------------- | ------- |
| 0            | Id               | TEXT    |
| 1            | Name             | TEXT    |
| 2            | Pricebook2Id     | TEXT    |
| 3            | Product2Id       | TEXT    |
| 4            | UnitPrice        | REAL    |
| 5            | IsActive         | INTEGER |
| 6            | UseStandardPrice | INTEGER |
| 7            | CreatedDate      | TEXT    |
| 8            | CreatedById      | TEXT    |
| 9            | LastModifiedDate | TEXT    |
| 10           | LastModifiedById | TEXT    |
| 11           | SystemModstamp   | TEXT    |
| 12           | ProductCode      | TEXT    |
| 13           | IsDeleted        | INTEGER |
| 14           | IsArchived       | INTEGER |

### salesforce_quote

Represents a quote, which is a record showing proposed prices for products and services.
Quotes can be created from and synced with opportunities, and emailed as PDFs to customers.

| Column index | Column name               | type    |
| ------------ | ------------------------- | ------- |
| 0            | Id                        | TEXT    |
| 1            | OwnerId                   | TEXT    |
| 2            | IsDeleted                 | INTEGER |
| 3            | Name                      | TEXT    |
| 4            | CreatedDate               | TEXT    |
| 5            | CreatedById               | TEXT    |
| 6            | LastModifiedDate          | TEXT    |
| 7            | LastModifiedById          | TEXT    |
| 8            | SystemModstamp            | TEXT    |
| 9            | LastViewedDate            | TEXT    |
| 10           | LastReferencedDate        | TEXT    |
| 11           | OpportunityId             | TEXT    |
| 12           | Pricebook2Id              | TEXT    |
| 13           | ContactId                 | TEXT    |
| 14           | QuoteNumber               | TEXT    |
| 15           | IsSyncing                 | INTEGER |
| 16           | ShippingHandling          | REAL    |
| 17           | Tax                       | REAL    |
| 18           | Status                    | TEXT    |
| 19           | ExpirationDate            | TEXT    |
| 20           | Description               | TEXT    |
| 21           | Subtotal                  | REAL    |
| 22           | TotalPrice                | REAL    |
| 23           | LineItemCount             | REAL    |
| 24           | BillingStreet             | TEXT    |
| 25           | BillingCity               | TEXT    |
| 26           | BillingState              | TEXT    |
| 27           | BillingPostalCode         | TEXT    |
| 28           | BillingCountry            | TEXT    |
| 29           | BillingLatitude           | REAL    |
| 30           | BillingLongitude          | REAL    |
| 31           | BillingGeocodeAccuracy    | TEXT    |
| 32           | BillingAddress            | TEXT    |
| 33           | ShippingStreet            | TEXT    |
| 34           | ShippingCity              | TEXT    |
| 35           | ShippingState             | TEXT    |
| 36           | ShippingPostalCode        | TEXT    |
| 37           | ShippingCountry           | TEXT    |
| 38           | ShippingLatitude          | REAL    |
| 39           | ShippingLongitude         | REAL    |
| 40           | ShippingGeocodeAccuracy   | TEXT    |
| 41           | ShippingAddress           | TEXT    |
| 42           | QuoteToStreet             | TEXT    |
| 43           | QuoteToCity               | TEXT    |
| 44           | QuoteToState              | TEXT    |
| 45           | QuoteToPostalCode         | TEXT    |
| 46           | QuoteToCountry            | TEXT    |
| 47           | QuoteToLatitude           | REAL    |
| 48           | QuoteToLongitude          | REAL    |
| 49           | QuoteToGeocodeAccuracy    | TEXT    |
| 50           | QuoteToAddress            | TEXT    |
| 51           | AdditionalStreet          | TEXT    |
| 52           | AdditionalCity            | TEXT    |
| 53           | AdditionalState           | TEXT    |
| 54           | AdditionalPostalCode      | TEXT    |
| 55           | AdditionalCountry         | TEXT    |
| 56           | AdditionalLatitude        | REAL    |
| 57           | AdditionalLongitude       | REAL    |
| 58           | AdditionalGeocodeAccuracy | TEXT    |
| 59           | AdditionalAddress         | TEXT    |
| 60           | BillingName               | TEXT    |
| 61           | ShippingName              | TEXT    |
| 62           | QuoteToName               | TEXT    |
| 63           | AdditionalName            | TEXT    |
| 64           | Email                     | TEXT    |
| 65           | Phone                     | TEXT    |
| 66           | Fax                       | TEXT    |
| 67           | ContractId                | TEXT    |
| 68           | AccountId                 | TEXT    |
| 69           | Discount                  | REAL    |
| 70           | GrandTotal                | REAL    |
| 71           | CanCreateQuoteLineItems   | INTEGER |
| 72           | RelatedWorkId             | TEXT    |

### salesforce_quotelineitem

Represents a quote line item, which is a member of the list of Product2 products associated with a Quote, along with other information about those line items on that quote.

A Quote record can have QuoteLineItem records only if the Quote has a Pricebook2. A QuoteLineItem must correspond to a Product2 that is listed in the quote's Pricebook2.

| Column index | Column name           | type    |
| ------------ | --------------------- | ------- |
| 0            | Id                    | TEXT    |
| 1            | IsDeleted             | INTEGER |
| 2            | LineNumber            | TEXT    |
| 3            | CreatedDate           | TEXT    |
| 4            | CreatedById           | TEXT    |
| 5            | LastModifiedDate      | TEXT    |
| 6            | LastModifiedById      | TEXT    |
| 7            | SystemModstamp        | TEXT    |
| 8            | LastViewedDate        | TEXT    |
| 9            | LastReferencedDate    | TEXT    |
| 10           | QuoteId               | TEXT    |
| 11           | PricebookEntryId      | TEXT    |
| 12           | OpportunityLineItemId | TEXT    |
| 13           | Quantity              | REAL    |
| 14           | UnitPrice             | REAL    |
| 15           | Discount              | REAL    |
| 16           | Description           | TEXT    |
| 17           | ServiceDate           | TEXT    |
| 18           | Product2Id            | TEXT    |
| 19           | SortOrder             | REAL    |
| 20           | ListPrice             | REAL    |
| 21           | Subtotal              | REAL    |
| 22           | TotalPrice            | REAL    |

### salesforce_order

Represents an order associated with a contract or an account.

The Status field specifies the current state of an order. Status strings represent its current state (Draft or Activated).

When you create an order, the Status Code must be Draft and the Status must be any value that corresponds to a Status Code of Draft. The application can then activate an order by updating it and setting the value in its Status field to an Activated state; however, the Status field is the only field you can update when activating the order.

After an order is activated, you can change the Status back to the Draft state—but only if the order doesn’t have any child reduction order products. You can delete orders when the Status is Draft but not when its Status is Activated.

| Column index | Column name             | type    |
| ------------ | ----------------------- | ------- |
| 0            | Id                      | TEXT    |
| 1            | OwnerId                 | TEXT    |
| 2            | ContractId              | TEXT    |
| 3            | AccountId               | TEXT    |
| 4            | Pricebook2Id            | TEXT    |
| 5            | OriginalOrderId         | TEXT    |
| 6            | EffectiveDate           | TEXT    |
| 7            | EndDate                 | TEXT    |
| 8            | IsReductionOrder        | INTEGER |
| 9            | Status                  | TEXT    |
| 10           | Description             | TEXT    |
| 11           | CustomerAuthorizedById  | TEXT    |
| 12           | CustomerAuthorizedDate  | TEXT    |
| 13           | CompanyAuthorizedById   | TEXT    |
| 14           | CompanyAuthorizedDate   | TEXT    |
| 15           | Type                    | TEXT    |
| 16           | BillingStreet           | TEXT    |
| 17           | BillingCity             | TEXT    |
| 18           | BillingState            | TEXT    |
| 19           | BillingPostalCode       | TEXT    |
| 20           | BillingCountry          | TEXT    |
| 21           | BillingLatitude         | REAL    |
| 22           | BillingLongitude        | REAL    |
| 23           | BillingGeocodeAccuracy  | TEXT    |
| 24           | BillingAddress          | TEXT    |
| 25           | ShippingStreet          | TEXT    |
| 26           | ShippingCity            | TEXT    |
| 27           | ShippingState           | TEXT    |
| 28           | ShippingPostalCode      | TEXT    |
| 29           | ShippingCountry         | TEXT    |
| 30           | ShippingLatitude        | REAL    |
| 31           | ShippingLongitude       | REAL    |
| 32           | ShippingGeocodeAccuracy | TEXT    |
| 33           | ShippingAddress         | TEXT    |
| 34           | Name                    | TEXT    |
| 35           | PoDate                  | TEXT    |
| 36           | PoNumber                | TEXT    |
| 37           | OrderReferenceNumber    | TEXT    |
| 38           | BillToContactId         | TEXT    |
| 39           | ShipToContactId         | TEXT    |
| 40           | ActivatedDate           | TEXT    |
| 41           | ActivatedById           | TEXT    |
| 42           | StatusCode              | TEXT    |
| 43           | OrderNumber             | TEXT    |
| 44           | TotalAmount             | REAL    |
| 45           | CreatedDate             | TEXT    |
| 46           | CreatedById             | TEXT    |
| 47           | LastModifiedDate        | TEXT    |
| 48           | LastModifiedById        | TEXT    |
| 49           | IsDeleted               | INTEGER |
| 50           | SystemModstamp          | TEXT    |
| 51           | LastViewedDate          | TEXT    |
| 52           | LastReferencedDate      | TEXT    |

### salesforce_orderitem

Represents an order product that your organization sells.

An order can have associated order product records only if the order has a price book associated with it. An order product must correspond to a product that is listed in the order’s price book.

| Column index | Column name         | type    |
| ------------ | ------------------- | ------- |
| 0            | Id                  | TEXT    |
| 1            | Product2Id          | TEXT    |
| 2            | IsDeleted           | INTEGER |
| 3            | OrderId             | TEXT    |
| 4            | PricebookEntryId    | TEXT    |
| 5            | OriginalOrderItemId | TEXT    |
| 6            | AvailableQuantity   | REAL    |
| 7            | Quantity            | REAL    |
| 8            | UnitPrice           | REAL    |
| 9            | ListPrice           | REAL    |
| 10           | TotalPrice          | REAL    |
| 11           | ServiceDate         | TEXT    |
| 12           | EndDate             | TEXT    |
| 13           | Description         | TEXT    |
| 14           | CreatedDate         | TEXT    |
| 15           | CreatedById         | TEXT    |
| 16           | LastModifiedDate    | TEXT    |
| 17           | LastModifiedById    | TEXT    |
| 18           | SystemModstamp      | TEXT    |
| 19           | OrderItemNumber     | TEXT    |

### salesforce_invoice

Represents a financial document describing the total amount a buyer must pay for goods or services provided.
Users can edit non-posted invoices. Posted invoices can’t be deleted. After an invoice is posted, users can make payments against it to reduce its balance.

To access these entities, your org must have a Salesforce Order Management or D2C Commerce license.

| Column index | Column name                  | type    |
| ------------ | ---------------------------- | ------- |
| 0            | Id                           | TEXT    |
| 1            | OwnerId                      | TEXT    |
| 2            | IsDeleted                    | INTEGER |
| 3            | DocumentNumber               | TEXT    |
| 4            | CreatedDate                  | TEXT    |
| 5            | CreatedById                  | TEXT    |
| 6            | LastModifiedDate             | TEXT    |
| 7            | LastModifiedById             | TEXT    |
| 8            | SystemModstamp               | TEXT    |
| 9            | LastViewedDate               | TEXT    |
| 10           | LastReferencedDate           | TEXT    |
| 11           | ReferenceEntityId            | TEXT    |
| 12           | InvoiceNumber                | TEXT    |
| 13           | BillingAccountId             | TEXT    |
| 14           | TotalAmount                  | REAL    |
| 15           | TotalAmountWithTax           | REAL    |
| 16           | TotalChargeAmount            | REAL    |
| 17           | TotalAdjustmentAmount        | REAL    |
| 18           | TotalTaxAmount               | REAL    |
| 19           | Status                       | TEXT    |
| 20           | InvoiceDate                  | TEXT    |
| 21           | DueDate                      | TEXT    |
| 22           | BillToContactId              | TEXT    |
| 23           | Description                  | TEXT    |
| 24           | Balance                      | REAL    |
| 25           | TotalChargeTaxAmount         | REAL    |
| 26           | TotalChargeAmountWithTax     | REAL    |
| 27           | TotalAdjustmentTaxAmount     | REAL    |
| 28           | TotalAdjustmentAmountWithTax | REAL    |
| 29           | NetCreditsApplied            | REAL    |
| 30           | NetPaymentsApplied           | REAL    |
| 31           | IsInvoiceLocked              | INTEGER |
| 32           | InvoiceLockedDateTime        | TEXT    |

### salesforce_invoiceline

Represents the amount that a buyer must pay for a product, service, or fee. Invoice lines are created based on the amount of an order line.

This table is available when Order Management or Subscription Management is enabled.

| Column index | Column name                 | type    |
| ------------ | --------------------------- | ------- |
| 0            | Id                          | TEXT    |
| 1            | IsDeleted                   | INTEGER |
| 2            | Name                        | TEXT    |
| 3            | CreatedDate                 | TEXT    |
| 4            | CreatedById                 | TEXT    |
| 5            | LastModifiedDate            | TEXT    |
| 6            | LastModifiedById            | TEXT    |
| 7            | SystemModstamp              | TEXT    |
| 8            | InvoiceId                   | TEXT    |
| 9            | ReferenceEntityItemId       | TEXT    |
| 10           | GroupReferenceEntityItemId  | TEXT    |
| 11           | LineAmount                  | REAL    |
| 12           | Quantity                    | REAL    |
| 13           | UnitPrice                   | REAL    |
| 14           | ChargeAmount                | REAL    |
| 15           | TaxAmount                   | REAL    |
| 16           | AdjustmentAmount            | REAL    |
| 17           | InvoiceStatus               | TEXT    |
| 18           | Description                 | TEXT    |
| 19           | InvoiceLineStartDate        | TEXT    |
| 20           | InvoiceLineEndDate          | TEXT    |
| 21           | ReferenceEntityItemType     | TEXT    |
| 22           | ReferenceEntityItemTypeCode | TEXT    |
| 23           | Product2Id                  | TEXT    |
| 24           | RelatedLineId               | TEXT    |
| 25           | Type                        | TEXT    |
| 26           | TaxName                     | TEXT    |
| 27           | TaxCode                     | TEXT    |
| 28           | TaxRate                     | REAL    |
| 29           | TaxEffectiveDate            | TEXT    |
| 30           | ChargeTaxAmount             | REAL    |
| 31           | ChargeAmountWithTax         | REAL    |
| 32           | AdjustmentTaxAmount         | REAL    |
| 33           | AdjustmentAmountWithTax     | REAL    |
| 34           | TaxProcessingStatus         | TEXT    |

The Salesforce plugin provides tables for the following objects: `account`, `contact`, `lead`, `opportunity`, `case`, `task`, `event`, `campaign`, `user`, `campaignmember`, `asset`, `contract`, `contractlineitem`, `servicecontract`, `solution`, `pricebook2`, `product2`, `productitem`, `pricebookentry`, `quote`, `quotelineitem`, `order`, `orderitem`, `invoice`, `invoiceline`, `report`, `dashboard`, `document`, `payment`, `paymentlineinvoice`

### salesforce_report

Represents a report, a set of data that meets certain criteria, displayed in an organized way. Access is read-only.

| Column index | Column name        | type    |
| ------------ | ------------------ | ------- |
| 0            | Id                 | TEXT    |
| 1            | OwnerId            | TEXT    |
| 2            | FolderName         | TEXT    |
| 3            | CreatedDate        | TEXT    |
| 4            | CreatedById        | TEXT    |
| 5            | LastModifiedDate   | TEXT    |
| 6            | LastModifiedById   | TEXT    |
| 7            | IsDeleted          | INTEGER |
| 8            | Name               | TEXT    |
| 9            | Description        | TEXT    |
| 10           | DeveloperName      | TEXT    |
| 11           | NamespacePrefix    | TEXT    |
| 12           | LastRunDate        | TEXT    |
| 13           | SystemModstamp     | TEXT    |
| 14           | Format             | TEXT    |
| 15           | LastViewedDate     | TEXT    |
| 16           | LastReferencedDate | TEXT    |

### salesforce_dashboard

Represents a dashboard, which shows data from custom reports as visual components. Access is read-only.

| Column index | Column name                  | type    |
| ------------ | ---------------------------- | ------- |
| 0            | Id                           | TEXT    |
| 1            | IsDeleted                    | INTEGER |
| 2            | OwnerId                      | TEXT    |
| 3            | FolderId                     | TEXT    |
| 4            | FolderName                   | TEXT    |
| 5            | Title                        | TEXT    |
| 6            | DeveloperName                | TEXT    |
| 7            | NamespacePrefix              | TEXT    |
| 8            | Description                  | TEXT    |
| 9            | LeftSize                     | TEXT    |
| 10           | MiddleSize                   | TEXT    |
| 11           | RightSize                    | TEXT    |
| 12           | CreatedDate                  | TEXT    |
| 13           | CreatedById                  | TEXT    |
| 14           | LastModifiedDate             | TEXT    |
| 15           | LastModifiedById             | TEXT    |
| 16           | SystemModstamp               | TEXT    |
| 17           | RunningUserId                | TEXT    |
| 18           | TitleColor                   | REAL    |
| 19           | TitleSize                    | REAL    |
| 20           | TextColor                    | REAL    |
| 21           | BackgroundStart              | REAL    |
| 22           | BackgroundEnd                | REAL    |
| 23           | BackgroundDirection          | TEXT    |
| 24           | Type                         | TEXT    |
| 25           | LastViewedDate               | TEXT    |
| 26           | LastReferencedDate           | TEXT    |
| 27           | DashboardResultRefreshedDate | TEXT    |
| 28           | DashboardResultRunningUser   | TEXT    |
| 29           | ColorPalette                 | TEXT    |
| 30           | ChartTheme                   | TEXT    |

### salesforce_document

Represents a file that a user has uploaded. Unlike Attachment records, documents are not attached to a parent object.

When creating or updating a document, you can specify a value in either the Body or Url fields, but not both.

| Column index | Column name        | type    |
| ------------ | ------------------ | ------- |
| 0            | Id                 | TEXT    |
| 1            | FolderId           | TEXT    |
| 2            | IsDeleted          | INTEGER |
| 3            | Name               | TEXT    |
| 4            | DeveloperName      | TEXT    |
| 5            | NamespacePrefix    | TEXT    |
| 6            | ContentType        | TEXT    |
| 7            | Type               | TEXT    |
| 9            | BodyLength         | REAL    |
| 10           | Body               | TEXT    |
| 11           | Url                | TEXT    |
| 12           | Description        | TEXT    |
| 13           | Keywords           | TEXT    |
| 14           | IsInternalUseOnly  | INTEGER |
| 15           | AuthorId           | TEXT    |
| 16           | CreatedDate        | TEXT    |
| 17           | CreatedById        | TEXT    |
| 18           | LastModifiedDate   | TEXT    |
| 19           | LastModifiedById   | TEXT    |
| 20           | SystemModstamp     | TEXT    |
| 21           | IsBodySearchable   | INTEGER |
| 22           | LastViewedDate     | TEXT    |
| 23           | LastReferencedDate | TEXT    |

### salesforce_payment

Represents a single event when a shopper makes a payment. For credit cards, this event is a payment capture or payment sale, but it doesn't appear on the shopper's credit card statement.

| Column index | Column name                   | type    |
| ------------ | ----------------------------- | ------- |
| 0            | Id                            | TEXT    |
| 1            | IsDeleted                     | INTEGER |
| 2            | PaymentNumber                 | TEXT    |
| 3            | CreatedDate                   | TEXT    |
| 4            | CreatedById                   | TEXT    |
| 5            | LastModifiedDate              | TEXT    |
| 6            | LastModifiedById              | TEXT    |
| 7            | SystemModstamp                | TEXT    |
| 8            | LastViewedDate                | TEXT    |
| 9            | LastReferencedDate            | TEXT    |
| 10           | PaymentGroupId                | TEXT    |
| 11           | AccountId                     | TEXT    |
| 12           | PaymentAuthorizationId        | TEXT    |
| 13           | Date                          | TEXT    |
| 14           | CancellationDate              | TEXT    |
| 15           | Amount                        | REAL    |
| 16           | Status                        | TEXT    |
| 17           | Type                          | TEXT    |
| 18           | ProcessingMode                | TEXT    |
| 19           | GatewayRefNumber              | TEXT    |
| 20           | ClientContext                 | TEXT    |
| 21           | GatewayResultCode             | TEXT    |
| 22           | SfResultCode                  | TEXT    |
| 23           | GatewayDate                   | TEXT    |
| 24           | CancellationGatewayRefNumber  | TEXT    |
| 25           | CancellationGatewayResultCode | TEXT    |
| 26           | CancellationSfResultCode      | TEXT    |
| 27           | CancellationGatewayDate       | TEXT    |
| 28           | Comments                      | TEXT    |
| 29           | ImpactAmount                  | REAL    |
| 30           | EffectiveDate                 | TEXT    |
| 31           | CancellationEffectiveDate     | TEXT    |
| 32           | GatewayResultCodeDescription  | TEXT    |
| 33           | GatewayRefDetails             | TEXT    |
| 34           | IpAddress                     | TEXT    |
| 35           | MacAddress                    | TEXT    |
| 36           | Phone                         | TEXT    |
| 37           | Email                         | TEXT    |
| 38           | PaymentGatewayId              | TEXT    |
| 39           | PaymentMethodId               | TEXT    |
| 40           | TotalApplied                  | REAL    |
| 41           | TotalUnapplied                | REAL    |
| 42           | NetApplied                    | REAL    |
| 43           | Balance                       | REAL    |
| 44           | TotalRefundApplied            | REAL    |
| 45           | TotalRefundUnapplied          | REAL    |
| 46           | NetRefundApplied              | REAL    |
| 47           | PaymentIntentGuid             | TEXT    |

### salesforce_paymentlineinvoice

Represents a payment allocated to or unallocated from an invoice. 

To access Commerce Payments entities, your org must have a Salesforce Order Management license with the Payment Platform org permission activated. Commerce Payments entities are available only in Lightning Experience.

Use a payment line to apply all or part of a payment’s balance to an invoice. The PaymentLineInvoice object represents the balance taken from the payment and applied toward the invoice. You can apply a payment’s balance when you create the payment record or afterward. The payment line must have the same currency as the parent payment.

A payment line has an amount, which represents the total amount taken from the payment, and balance, which represents the remaining amount after the payment line has been applied to an invoice. A payment’s amount can’t be less than the sum of all of its payment line amounts.

One payment can have multiple payment lines. A payment line must be related to only payment.

You can create multiple payment lines on a payment apply each line to different invoices on the same account, or to invoices on different accounts.

![Schema of paymentlineinvoice](https://developer.salesforce.com/docs/resources/img/en-us/252.0?doc_id=dev_guides%2Fapi%2Fimages%2FSforce_payment_flow.png&folder=object_reference)

| Column index | Column name              | type    |
| ------------ | ------------------------ | ------- |
| 0            | Id                       | TEXT    |
| 1            | IsDeleted                | INTEGER |
| 2            | PaymentLineInvoiceNumber | TEXT    |
| 3            | CreatedDate              | TEXT    |
| 4            | CreatedById              | TEXT    |
| 5            | LastModifiedDate         | TEXT    |
| 6            | LastModifiedById         | TEXT    |
| 7            | SystemModstamp           | TEXT    |
| 8            | LastViewedDate           | TEXT    |
| 9            | LastReferencedDate       | TEXT    |
| 10           | InvoiceId                | TEXT    |
| 11           | PaymentId                | TEXT    |
| 12           | Amount                   | REAL    |
| 13           | Type                     | TEXT    |
| 14           | HasBeenUnapplied         | TEXT    |
| 15           | Comments                 | TEXT    |
| 16           | Date                     | TEXT    |
| 17           | AppliedDate              | TEXT    |
| 18           | EffectiveDate            | TEXT    |
| 19           | UnappliedDate            | TEXT    |
| 20           | AssociatedAccountId      | TEXT    |
| 21           | AssociatedPaymentLineId  | TEXT    |
| 22           | ImpactAmount             | REAL    |
| 23           | EffectiveImpactAmount    | REAL    |
| 24           | PaymentBalance           | REAL    |