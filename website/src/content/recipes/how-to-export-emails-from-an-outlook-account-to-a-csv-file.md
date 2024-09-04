---
title: "How to export emails from an Outlook account to a CSV file?"
description: "Learn how to seamlessly export emails from your Outlook account to a CSV file using Anyquery, covering installation, setting up profiles, and executing SQL queries."
---

## How to Export Emails from an Outlook Account to a CSV File

Anyquery is a SQL query engine that allows you to run SQL queries on virtually anything, including email accounts. In this tutorial, we will show you how to export emails from an Outlook account to a CSV file using Anyquery.

## Prerequisites

Before starting, ensure you have the following:

- A working [installation of Anyquery](https://anyquery.dev/docs/#installation)
- A working Outlook account
- The IMAP plugin installed

## Step 1: Install the IMAP Plugin

First, install the IMAP plugin if you haven't already:

```bash
anyquery install imap
```

## Step 2: Create an Application Password

Go to [https://account.live.com/proofs/manage](https://account.live.com/proofs/manage) and create an application password for Anyquery. Input the name you want and copy the generated password (only if using two-factor authentication).

## Step 3: Create a New Profile for the IMAP Plugin

Now, create a new profile for the IMAP plugin. Run the following command:

```bash
anyquery profile new default imap outlook
```

Fill in the following details when prompted:

- **Host**: `outlook.office365.com`
- **Port**: `993`
- **Username**: Your Outlook email address
- **Password**: The application password you generated in Step 2 or your regular password if not using two-factor authentication

## Step 4: Export Emails to a CSV File

Now that you have set up the profile, you can export your emails to a CSV file using SQL. Here is an example command to get the subject, sender, and date of the first 100 emails and export them to a CSV file:

```bash
anyquery -q "SELECT subject, _from, received_at FROM outlook_imap_emails LIMIT 100" --csv > emails.csv
```

:::caution
Ensure you include the `LIMIT` clause to avoid fetching an overwhelming number of emails.
:::

Additionally, you can customize your export by filtering specific emails. For example, to export only unread emails:

```bash
anyquery -q "SELECT subject, _from, received_at FROM outlook_imap_emails WHERE flags NOT LIKE '%\"Seen\"%' LIMIT 100" --csv > unread_emails.csv
```

## Conclusion

You have successfully exported emails from your Outlook account to a CSV file using Anyquery. For more information on available functions and further customization, refer to the [functions documentation](https://anyquery.dev/docs/reference/functions).


