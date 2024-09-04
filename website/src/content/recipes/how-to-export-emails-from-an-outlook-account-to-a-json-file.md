---
title: "How to export emails from an Outlook account to a JSON file?"
description: "Learn to export emails from an Outlook account to a JSON file using Anyquery. Follow steps to create an app password, install the IMAP plugin, query, and export emails."
---

# How to Export Emails from an Outlook Account to a JSON File

Anyquery is a SQL query engine that enables you to execute SQL queries on pretty much anything, including your email accounts via the IMAP protocol. In this tutorial, we will guide you on how to export emails from an Outlook account to a JSON file.

## Prerequisites

Before starting, ensure you have the following:

- A working [installation of Anyquery](https://anyquery.dev/docs/#installation)
- A Gmail account
- The IMAP plugin installed (`anyquery install imap`)
- An application password for Outlook IMAP access

## Step 1: Create an Application Password

To connect Anyquery to your Outlook account, you will need to create an application password (only if using two-factor authentication). Follow these steps:

1. Go to [Outlook security settings](https://myaccount.microsoft.com/security).
2. Under the section `App passwords`, generate a new app password.
3. Note the password; you will need it in the next step.

## Step 2: Install the IMAP Plugin

If you haven't already installed the IMAP plugin, you can do so by running:

```bash
anyquery install imap
```

## Step 3: Create a New Profile for the IMAP Plugin

Next, create a new profile for the IMAP plugin to authenticate with your Outlook account:

```bash
anyquery profile new default imap outlook
```

Fill in the following details when prompted:

- **Host**: `imap-mail.outlook.com`
- **Port**: `993`
- **Username**: Your Outlook email address
- **Password**: The application password you generated in Step 1 or your regular password if not using two-factor authentication

## Step 4: Query Your Emails

Once the profile is set up, you can query your emails using SQL. For example, to get the subject and the sender of the first 10 emails, run the following command:

```sql
SELECT * FROM outlook_imap_emails LIMIT 10;
```

## Step 5: Export Emails to a JSON File

Now, you can export your emails to a JSON file using Anyquery. The following command exports the first 100 emails to a JSON file named `emails.json`:

```bash
anyquery -q "SELECT * FROM outlook_imap_emails LIMIT 100" --json > emails.json
```

### Example Queries

1. **Exporting emails with specific subjects:**

   ```bash
   anyquery -q "SELECT * FROM outlook_imap_emails WHERE subject LIKE '%Meeting%' LIMIT 50" --json > meeting_emails.json
   ```

2. **Exporting unread emails:**

   ```bash
   anyquery -q "SELECT * FROM outlook_imap_emails WHERE flags NOT LIKE '%\"Seen\"%' LIMIT 100" --json > unread_emails.json
   ```

3. **Exporting emails from a specific sender:**

   ```bash
   anyquery -q "SELECT * FROM outlook_imap_emails WHERE _from LIKE '%@example.com%' LIMIT 100" --json > example_com_emails.json
   ```

## Conclusion

You've successfully learned how to export emails from your Outlook account to a JSON file using Anyquery. Now you can explore your email data further by running custom queries.

For more information on the IMAP plugin and its capabilities, refer to the [IMAP plugin documentation](https://anyquery.dev/integrations/imap). For troubleshooting, visit the [Anyquery troubleshooting page](https://anyquery.dev/docs/usage/troubleshooting/).
