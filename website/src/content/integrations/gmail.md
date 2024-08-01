---
title: Gmail
description: Query your Gmail emails using SQL
icon: https://cdn.jsdelivr.net/gh/walkxcode/dashboard-icons@main/svg/gmail.svg
---

Anyquery is a query engine that allows you to run SQL queries on pretty much anything. In this guide, we will connect Anyquery to the Gmail IMAP server and query your emails using SQL.

## Prerequisites

Before starting, ensure you have the following:

- A working [installation of Anyquery](/docs/#installation)
- A Gmail account
- The IMAP plugin installed [tutorial](/integrations/imap)

## Step 1: Create an application password

Go to [https://myaccount.google.com/apppasswords](https://myaccount.google.com/apppasswords) and create an application password for Anyquery. Input the name you want and copy the generated password.

## Step 2: Create a new profile for the IMAP plugin

First, let's create a new profile for the IMAP plugin. Run the following command:

```bash title="Install the plugin and create a new profile"
# If the plugin is not installed
anyquery install imap
# Otherwise, create a new profile
anyquery profile new default imap myprofile
```

Fill in the following details:

- Host: `imap.gmail.com`
- Port: `993`
- Username: Your Gmail email
- Password: The application password you generated without the spaces

## Step 3: Query your emails

Now that you have set up the profile, you can query your emails using SQL. Here is an example query to get the subject and the sender of the first 10 emails:

```sql title="Query your emails"
-- If the plugin is not installed
SELECT * FROM imap_emails LIMIT 10;
-- Otherwise, use the profile
SELECT * FROM myprofile_imap_emails LIMIT 10;
```
