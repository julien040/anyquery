# Calendar plugin

Query events from a calendar (iCal format) with SQL.

## Setup

```bash
anyquery install calendar
```

## Usage

```sql
-- Select all events during the year 2021
SELECT * FROM calendar_events('./my_calendar.ics') WHERE start_at > '2021-01-01' AND end_at < '2021-12-31';
-- Select all events where the summary contains 'meeting'
SELECT * FROM calendar_events('./my_calendar.ics') WHERE summary LIKE '%meeting%';
-- Select all events where the description contains 'meeting' and the location is 'Room A113'
SELECT * FROM calendar_events WHERE description LIKE '%meeting%' AND location = 'Room A113' AND path = './my_calendar.ics';
-- Select all events from a remote calendar (Google Calendar for example)
SELECT * FROM calendar_events('https://calendar.google.com/calendar/ical/mymail%40gmail.com/private-ba23e19e/basic.ics');
```

### Sharing a Google Calendar

Go to the [Google Calendar settings](https://calendar.google.com/calendar/r/settings/calendar) and click on the calendar you want to share. Then, click on the `Integrate calendar` section and copy the `Secret address in iCal format`. You can then use this URL to query the calendar with `SELECT * FROM calendar_events('https://my-secret-adress-that-I-copied.ics')`.

### Sharing an Apple Calendar

Click on File > Export > Export. Then, save the file and use the path to query the calendar with `SELECT * FROM calendar_events('./my-calendar-export.ics')`.

### Sharing a Microsoft Outlook Calendar online

Go to the [Outlook calendar settings](https://outlook.live.com/calendar/0/options/calendar/SharedCalendars) and click on the calendar you want to share. Then, click on the `Publish calendar` section and copy the `Publish to web` link that ends with `.ics`. You can then use this URL to query the calendar with `SELECT * FROM calendar_events('https://my-secret-adress-that-I-copied.ics')`.

## Schema

Not all fields are guaranteed to be present in the table. Feel free to open an issue if you need a specific field.

| Column index | Column name      | type    |
| ------------ | ---------------- | ------- |
| 0            | id               | TEXT    |
| 1            | start_at         | TEXT    |
| 2            | end_at           | TEXT    |
| 3            | summary          | TEXT    |
| 4            | description      | TEXT    |
| 5            | attendees        | TEXT    |
| 6            | status           | TEXT    |
| 7            | priority         | TEXT    |
| 8            | location         | TEXT    |
| 9            | geo              | TEXT    |
| 10           | organizer        | TEXT    |
| 11           | sequence         | INTEGER |
| 12           | created_at       | TEXT    |
| 13           | last_modified_at | TEXT    |

## Limitations

- The plugin only supports the iCal format.
- The plugin does not do any caching. It will read the file or download it every time you query the calendar. If you plan to query the calendar multiple times, you can import the data into a SQLite database with `CREATE TABLE my_calendar AS SELECT * FROM calendar_events('./my_calendar.ics')`.
