---
title: "How to export emails from a Gmail account to a SQLite database?"
description: "Learn how to export Gmail emails to a SQLite database using Anyquery. This guide covers prerequisites, setting up an IMAP profile, querying emails, and exporting data."
---

# How to Export Emails from a Gmail Account to a SQLite Database

Anyquery is a SQL query engine that allows you to run SQL queries on pretty much anything, including your Gmail emails. In this tutorial, we will show you how to export emails from a Gmail account to a SQLite database using Anyquery.

## Prerequisites

Before starting, ensure you have the following:

- A working [installation of Anyquery](https://anyquery.dev/docs/#installation)
- A Gmail account
- The IMAP plugin installed (`anyquery install imap`)

## Step 1: Create an Application Password

Gmail requires an application password for third-party apps to access your account. Follow these steps to create an application password:

1. Go to [Google Account Security](https://myaccount.google.com/security).
2. Under "Signing in to Google," click on **App passwords**.
3. You might need to sign in to your Google account again.
4. Select **Mail** as the app and **Other (Custom name)** as the device, then enter "Anyquery" and click **Generate**.
5. Copy the generated password. You will need it in the next step.

## Step 2: Create a New Profile for the IMAP Plugin

First, let's create a new profile for the IMAP plugin. Run the following commands:

```bash
# If the plugin is not installed
anyquery install imap

# Otherwise, create a new profile
anyquery profile new default imap gmail
```

Fill in the following details when prompted:

- **Host**: `imap.gmail.com`
- **Port**: `993`
- **Username**: Your Gmail email address
- **Password**: The application password you generated in Step 1

## Step 3: Query Your Emails

Now that you have set up the profile, you can query your emails using SQL. Here is an example query to get the subject and the sender of the first 10 emails:

```sql
SELECT * FROM gmail_imap_emails LIMIT 10;
```

## Step 4: Export Emails to a SQLite Database

To export emails to a SQLite database, follow these steps:

1. Open Anyquery in shell mode with a new SQLite database:

    ```bash
    anyquery -d emails.db
    ```

2. Create a table to store the emails:

    ```sql
    CREATE TABLE emails (
        uid INTEGER PRIMARY KEY,
        subject TEXT,
        sent_at TEXT,
        received_at TEXT,
        _from TEXT,
        to TEXT,
        reply_to TEXT,
        cc TEXT,
        bcc TEXT,
        message_id TEXT,
        flags TEXT,
        size INTEGER,
        folder TEXT
    );
    ```

3. Insert emails into the SQLite database:

    ```sql
    INSERT INTO emails (uid, subject, sent_at, received_at, _from, to, reply_to, cc, bcc, message_id, flags, size, folder)
    SELECT uid, subject, sent_at, received_at, _from, to, reply_to, cc, bcc, message_id, flags, size, folder
    FROM gmail_imap_emails;
    ```

## Conclusion

You have successfully exported emails from your Gmail account to a SQLite database using Anyquery. Now you can query and analyze your emails using SQL.

For more information on using the IMAP plugin, refer to the [IMAP plugin documentation](https://anyquery.dev/integrations/imap). For general troubleshooting, refer to the [troubleshooting documentation](https://anyquery.dev/docs/usage/troubleshooting/).
