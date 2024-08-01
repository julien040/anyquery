---
title: Outlook Calendar
description: Query events from your Outlook Calendar
icon: /icons/outlook.svg
---

Anyquery is a query engine that allows you to run SQL queries on pretty much anything. In this guide, we will connect Anyquery to Outlook Calendar using the Calendar plugin.

## Prerequisites

Before starting, ensure you have the following:

- A working [installation of Anyquery](/docs/#installation)
- The Calendar plugin installed [tutorial](/integrations/calendar) (`anyquery install calendar`)
- An Outlook account

## Get the iCalendar URL

First, you need to get the iCalendar URL of your Outlook Calendar. Follow these steps:

1. Go to [Outlook Calendar settings](https://outlook.live.com/calendar/options/calendar/SharedCalendars)
2. Scroll to the section `Publish a calendar`
3. Select the calendar you want to query
4. Choose `All calendar details` and click on `Publish`
5. Copy the `Ics` URL (the one that ends with `.ics`)

## Query your events

Now that you have the iCalendar URL, you can query your events using SQL. Here is an example query to get the summary and the start date of the first 10 events:

```sql
SELECT * FROM calendar_events('https://outlook.live.com/owa/calendar/00000000-0000-0000-0000-000000000000/00000000-0000-0000-0000-000000000000/calendar.ics') LIMIT 10;
```
