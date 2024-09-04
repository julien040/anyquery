---
title: "How to export emails from a Gmail account to a CSV file?"
description: "Learn to export Gmail emails to a CSV file using Anyquery with an IMAP plugin. Follow step-by-step instructions to configure, query, and export your data effortlessly."
---

# Export Emails from a Gmail Account to a CSV File

This tutorial will guide you through the process of exporting emails from a Gmail account to a CSV file using Anyquery. We will use the IMAP plugin to connect to Gmail and query the emails.

## Introduction to Anyquery

Anyquery allows you to write SQL queries on pretty much any data source. It is a query engine that can be used to query data from different sources like databases, APIs, and even files.

**Example**

```sql
-- List all your saved tracks from Spotify
SELECT * FROM spotify_saved_tracks;

-- Insert data from a git repository into a Google Sheet
INSERT INTO google_sheets_table (name, line_added) SELECT author_name, addition FROM git_commits_diff('https://github.com/vercel/next.js.git');
```

For more information on installation, refer to the [installation documentation](https://anyquery.dev/docs/#installation).

## Prerequisites

Before starting, ensure you have the following:

- A working installation of Anyquery
- A Gmail account
- The IMAP plugin installed

## Step 1: Install the IMAP Plugin

To interact with your Gmail account, we need to install the IMAP plugin. Run the following command:

```bash
anyquery install imap
```

## Step 2: Create an Application Password

Go to [Google Account Security](https://myaccount.google.com/apppasswords) and create an application password for Anyquery. Input the name you want and copy the generated password.

## Step 3: Configure the IMAP Plugin

Create a new profile for the IMAP plugin to connect to your Gmail account. Run the following command:

```bash
anyquery profile new default imap mygmail
```

Fill in the following details:

- **Host**: `imap.gmail.com`
- **Port**: `993`
- **Username**: Your Gmail email
- **Password**: The application password you generated without spaces

## Step 4: Query Emails from Gmail

Now that you have set up the profile, you can query your emails using SQL. Here is an example query to get the subject and the sender of the first 10 emails:

```sql
SELECT subject, _from FROM mygmail_imap_emails LIMIT 10;
```

## Step 5: Export Emails to CSV

To export the emails to a CSV file, you will use the `--csv` flag with Anyquery. Run the following command:

```bash
anyquery -q "SELECT subject, _from, received_at, body FROM mygmail_imap_emails" --csv > emails.csv
```

This command will save the emails in a file named `emails.csv`.

## Conclusion

You have successfully exported emails from your Gmail account to a CSV file using Anyquery. You can now explore and manipulate the exported data as needed. For further details on querying and exporting data, refer to the [functions documentation](https://anyquery.dev/docs/reference/functions).
