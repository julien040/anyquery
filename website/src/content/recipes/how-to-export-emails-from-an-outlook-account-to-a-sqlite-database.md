---
title: "How to export emails from an Outlook account to a SQLite database?"
description: "Learn how to export emails from Outlook to a SQLite database using Anyquery. Follow steps to set up IMAP, create a profile, and verify the export effortlessly."
---

# How to Export Emails from an Outlook Account to a SQLite Database

**Anyquery** is a SQL query engine that allows you to execute SQL queries on various data sources, including your Outlook emails. In this tutorial, we will guide you through exporting emails from your Outlook account to a SQLite database using Anyquery.

## Prerequisites

Before starting, ensure you have the following:

- A working [installation of Anyquery](https://anyquery.dev/docs/#installation)
- An Outlook account
- The IMAP plugin installed

## Step 1: Create an Application Password for Outlook (if using two-factor authentication)

1. Go to [https://account.microsoft.com/security](https://account.microsoft.com/security) and sign in with your Outlook account.
2. Under "More security options," select "Create a new app password."
3. Copy the generated password, as you'll need it in the next step

## Step 2: Install the IMAP Plugin and Create a Profile

First, install the IMAP plugin for Anyquery:

```bash
anyquery install imap
```

Next, create a new profile for the IMAP plugin:

```bash
anyquery profile new default imap outlook
```

You will be prompted to fill in the following details:

- **Host**: `imap-mail.outlook.com`
- **Port**: `993`
- **Username**: Your Outlook email address
- **Password**: The application password you generated in Step 1 or your regular password if not using two-factor authentication

## Step 3: Export Emails to a SQLite Database

Now that the profile is set up, you can export emails to a SQLite database.

1. First, start Anyquery in shell mode with the SQLite database:

```bash
anyquery my_emails.db
```

2. In the Anyquery shell, create a table to store the emails:

```sql
CREATE TABLE outlook_emails AS
SELECT * FROM outlook_imap_emails;
```

This command will fetch emails from your Outlook account and insert them into the `outlook_emails` table in your SQLite database.

## Step 4: Verify the Export

To verify that the emails have been exported correctly, you can run a simple query to check the contents of the `outlook_emails` table:

```sql
SELECT * FROM outlook_emails LIMIT 10;
```

This will display the first 10 emails from the `outlook_emails` table.

## Conclusion

You have successfully exported emails from your Outlook account to a SQLite database using Anyquery. You can now run SQL queries on your email data to analyze and manipulate it as needed.

For more information on querying and manipulating data with Anyquery, refer to the [official documentation](https://anyquery.dev/docs/usage/querying-files).

## Additional Resources

- [Installation Guide](https://anyquery.dev/docs/#installation)
- [IMAP Plugin Documentation](https://anyquery.dev/integrations/imap)
- [Troubleshooting](https://anyquery.dev/docs/usage/troubleshooting)
