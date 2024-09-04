---
title: "How to export emails from a Gmail account to a JSON file?"
description: "Learn how to export emails from your Gmail account to a JSON file using Anyquery. Follow steps to install the IMAP plugin, create application passwords, and run SQL queries."
---

# Export Emails from a Gmail Account to a JSON File

Anyquery is a SQL query engine that allows you to run SQL queries on pretty much anything. One of its strengths is the ability to query emails from your Gmail account and export the results to various formats, including JSON. In this tutorial, we will guide you through the process of exporting emails from your Gmail account to a JSON file.

## Prerequisites

Before starting, ensure you have the following:

- A working installation of Anyquery. You can find the installation instructions [here](https://anyquery.dev/docs/#installation).
- A Gmail account.
- The IMAP plugin for Anyquery installed.

## Step 1: Install the IMAP Plugin

First, you need to install the IMAP plugin for Anyquery. Run the following command:

```bash
anyquery install imap
```

## Step 2: Create an Application Password

Go to [https://myaccount.google.com/apppasswords](https://myaccount.google.com/apppasswords) and create an application password for Anyquery. Input the name you want and copy the generated password.

## Step 3: Create a New Profile for the IMAP Plugin

Next, create a new profile for the IMAP plugin. Run the following command:

```bash
anyquery profile new default imap gmail
```

The command will prompt you for the following details:
- **Host**: `imap.gmail.com`
- **Port**: `993`
- **Username**: Your Gmail email
- **Password**: The application password you generated earlier without the spaces

## Step 4: Query Your Emails

Now that you have set up the profile, you can query your emails using SQL. Here is an example query to get the subject and the sender of the first 10 emails:

```sql
SELECT * FROM gmail_imap_emails LIMIT 10;
```

## Step 5: Export Emails to a JSON File

To export emails from your Gmail account to a JSON file, you can use the following command:

```bash
anyquery -q "SELECT * FROM gmail_imap_emails" --json > emails.json
```

This command will query all the emails and export the result to a JSON file named `emails.json`.

## Conclusion

You have successfully exported emails from your Gmail account to a JSON file using Anyquery. You can now explore and analyze your emails in JSON format. For more information on the available functions and tables, refer to the [Anyquery documentation](https://anyquery.dev/docs/usage/troubleshooting/).

Feel free to experiment with different queries and export formats to suit your needs!

---

For more detailed information on how to use the IMAP plugin, refer to the [IMAP plugin documentation](https://anyquery.dev/integrations/imap).
