---
title: Google Calendar
description: Query events from your Google Calendar
icon: https://cdn.jsdelivr.net/gh/walkxcode/dashboard-icons@main/svg/google-calendar.svg
---

Anyquery is a query engine that allows you to run SQL queries on pretty much anything. In this guide, we will connect Anyquery to Google Calendar using the Calendar plugin.

## Prerequisites

Before starting, ensure you have the following:

- A working [installation of Anyquery](/docs/#installation)
- The Calendar plugin installed [tutorial](/integrations/calendar) (`anyquery install calendar`)
- A Google account

## Get the iCalendar URL

First, you need to get the iCalendar URL of your Google Calendar. Follow these steps:

1. Go to [Google Calendar settings](https://calendar.google.com/calendar/u/0/r/settings)
2. Click on the calendar you want to query in the left sidebar
3. Scroll down to the `Integrate calendar` section
4. Copy the `Secret address in iCal format` URL
5. It should look like `https://calendar.google.com/calendar/ical/.../private-.../basic.ics`

## Query your events

Now that you have the iCalendar URL, you can query your events using SQL. Here is an example query to get the summary and the start date of the first 10 events:

```sql
SELECT * FROM calendar_events('https://calendar.google.com/calendar/ical/.../private-.../basic.ics') LIMIT 10;
```
