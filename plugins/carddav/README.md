# CardDAV plugin

Query and manage CardDAV contacts with SQL.

## Usage

```sql
-- List all available address books
SELECT * FROM carddav_address_books;

-- Get all contacts from a CardDAV address book
SELECT * FROM carddav_contacts WHERE address_book = 'contacts/';

-- Search for contacts by name
SELECT full_name, email, phone FROM carddav_contacts 
WHERE address_book = 'contacts/' AND full_name LIKE '%John%';

-- Insert a new contact
INSERT INTO carddav_contacts (address_book, uid, full_name, email, phone) 
VALUES ('contacts/', 'unique-id-123', 'John Doe', 'john@example.com', '+1234567890');

-- Update a contact
UPDATE carddav_contacts 
SET email = 'newemail@example.com', organization = 'New Company'
WHERE address_book = 'contacts/' AND uid = 'unique-id-123';
```

## Installation

```bash
anyquery install carddav
```

Anyquery will ask you for your CardDAV server URL, username, and password during installation. Refer to the [guide](#popular-carddav-providers) below for more information on how to configure your CardDAV server.

### Popular CardDAV Providers

#### Nextcloud

```txt
URL: https://your-nextcloud.com/remote.php/dav/addressbooks/users/yourusername/
```

Create an app-specific password in Settings → Security → App passwords.

#### Google Contacts

Enable CardDAV API in Google Admin Console (for Workspace accounts). The Google Contacts API is not supported, but you can use Anyquery's integration for [Google Contacts](https://anyquery.dev/integrations/google_contacts).

#### Apple iCloud

```txt
URL: https://contacts.icloud.com/
Username: The email used by your Apple account
Password: Your Apple ID password
```

Use an app-specific password from Apple ID settings. Refer to [Apple's documentation](https://support.apple.com/en-au/102654#:~:text=Sign%20in%20to%20your%20Apple%20Account%20on%20account.apple.com,the%20steps%20on%20your%20screen.)

## Tables

### `carddav_address_books`

List available address books on the CardDAV server.

#### Schema

| Column index | Column name       | Type    | Description                         |
| ------------ | ----------------- | ------- | ----------------------------------- |
| 0            | path              | TEXT    | Address book path (use for queries) |
| 1            | name              | TEXT    | Display name of the address book    |
| 2            | description       | TEXT    | Description of the address book     |
| 3            | max_resource_size | INTEGER | Maximum resource size               |

### `carddav_contacts`

Query and manage contacts from CardDAV address books.

#### Schema

| Column index | Column name  | Type | Description                   |
| ------------ | ------------ | ---- | ----------------------------- |
| 0            | address_book | TEXT | Address book path (parameter) |
| 1            | uid          | TEXT | Unique identifier             |
| 2            | etag         | TEXT | ETag for conflict detection   |
| 3            | path         | TEXT | CardDAV resource path         |
| 4            | full_name    | TEXT | Full display name             |
| 5            | given_name   | TEXT | First name                    |
| 6            | family_name  | TEXT | Last name                     |
| 7            | middle_name  | TEXT | Middle name                   |
| 8            | prefix       | TEXT | Name prefix (Mr., Dr., etc.)  |
| 9            | suffix       | TEXT | Name suffix (Jr., Sr., etc.)  |
| 10           | nickname     | TEXT | Nickname                      |
| 11           | email        | TEXT | Primary email address         |
| 12           | home_email   | TEXT | Home email address            |
| 13           | work_email   | TEXT | Work email address            |
| 14           | other_email  | TEXT | Other email address           |
| 15           | emails       | TEXT | All emails (JSON array)       |
| 16           | phone        | TEXT | Primary phone number          |
| 17           | mobile_phone | TEXT | Mobile phone number           |
| 18           | work_phone   | TEXT | Work phone number             |
| 19           | organization | TEXT | Organization/Company          |
| 20           | title        | TEXT | Job title                     |
| 21           | role         | TEXT | Role/Position                 |
| 22           | birthday     | TEXT | Birthday (YYYY-MM-DD)         |
| 23           | anniversary  | TEXT | Anniversary (YYYY-MM-DD)      |
| 24           | note         | TEXT | Notes                         |
| 25           | url          | TEXT | Website URL                   |
| 26           | categories   | TEXT | Categories (JSON array)       |
| 27           | modified_at  | TEXT | Last modified timestamp       |

## Development

To develop and test the CardDAV plugin:

```bash
cd plugins/carddav
make
make test                # Run unit tests
make integration-test    # Run integration tests with real CardDAV server
```

For manual testing, start anyquery in dev mode and load the plugin:

```bash
anyquery --dev
```

```sql
SELECT load_dev_plugin('carddav', 'devManifest.json');
```

Configure your CardDAV credentials in `devManifest.json` before running tests. The test script will verify plugin functionality by listing address books, querying contacts, and testing insert/update operations.

## Limitations

- Address book creation and deletion are not supported yet
- Some CardDAV servers may have different URL formats or authentication requirements
- Large contact lists may take time to query due to CardDAV protocol limitations
- The plugin does not cache data - each query hits the CardDAV server directly
