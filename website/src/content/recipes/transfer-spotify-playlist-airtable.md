---
title: Transfer a Spotify playlist to Airtable
description: Learn how to transfer a Spotify playlist to Airtable using Anyquery.
---

Anyquery is a SQL query engine that enables you to execute SQL queries on virtually anything, including Spotify playlists. Moreover, as it can insert rows into Airtable, you can transfer a Spotify playlist to Airtable with a straightforward SQL query.

## Prerequisites

Let's start by creating a table in Airtable to store the Spotify playlist. First, create a new base in Airtable and add a table with the following columns:

- `track_name` (one line text)
- `artist_name` (one line text)
- `album_name` (one line text)
- `duration` (number)
- `track_popularity` (number)
- `explicit` (checkbox)

Note the `baseID` in the URL and the name of the table in the Navbar. You will need them later.

Before you start, you need to install Anyquery and authenticate with Spotify and Airtable.

- [Install Anyquery](/docs/#installation)
- [Authenticate with Spotify](/integrations/spotify)
- [Authenticate with Airtable](/integrations/airtable) and input the `baseID` and the name of the table.

Once done, check that both connections are working by running the following commands:

```bash
anyquery -q "SELECT * FROM spotify_saved_tracks LIMIT 1; SELECT * FROM airtable_table LIMIT 1"
```

## Transfer a Spotify playlist to Airtable

For this recipe, we will transfer `Today's Top Hits` to Airtable. You can replace `Today's Top Hits` with any other playlist. To find the playlist ID, share the playlist and copy the link. The ID is the last part of the URL.

For example, sharing the link of `Today's Top Hits` is `https://open.spotify.com/playlist/37i9dQZF1DXcBWIGoYBM5M?si=4204709c64cb49e7`. The playlist ID is `37i9dQZF1DXcBWIGoYBM5M`.

To transfer the playlist to Airtable, you can use the following SQL query:

```bash
anyquery -q "INSERT INTO airtable_table (track_name, artist_name, album_name, duration, track_popularity, explicit) SELECT track_name, artist_name ->> '$[0]' as artist_name, album_name, track_duration_ms, track_popularity, track_explicit FROM spotify_playlist('37i9dQZF1DXcBWIGoYBM5M')"
```

![Result](/images/docs/cVtNtBbd.png)

:::tip
You can insert up to 450 rows per minute due to the Airtable API rate limit. No worries, Anyquery will automatically wait for the next minute if you exceed the limit.
:::
