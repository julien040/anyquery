# Spotify plugin

This plugin allows you to run SQL queries on your Spotify data.

## Installation

To install this plugin, simply run the following command :

```bash
anyquery install spotify
```

## Configuration

1. Go to the [Spotify Developer Dashboard](https://developer.spotify.com/dashboard) and create a new application.
   1. Fill in whatever you want for the application name and description.
   2. Add `https://integration.anyquery.dev/spotify-result` to the Redirect URIs.
   3. Select Web API at the question "Which API/SDKs are you planning to use?".
   4. Accept the terms and conditions and click on the "Save" button.
    ![registration](https://cdn.jsdelivr.net/gh/julien040/anyquery@main/plugins/spotify/images/registration.png)
2. Click on settings in the top right hand corner and copy the Client ID and Client Secret (click on View client secret).
3. Go to the [AnyQuery Spotify plugin page](https://integration.anyquery.dev/spotify), fill in the Client ID and Client Secret and click on the "Submit" button.
    ![connect](https://cdn.jsdelivr.net/gh/julien040/anyquery@main/plugins/spotify/images/connect.png)
4. Click on the green button to connect your Spotify account.
   ![connect_spotify](https://cdn.jsdelivr.net/gh/julien040/anyquery@main/plugins/spotify/images/spotify-agree.png)
5. Copy your token and paste it in the configuration form.
   ![token](https://cdn.jsdelivr.net/gh/julien040/anyquery@main/plugins/spotify/images/token.png)

## Tables

The plugin provides the following tables:

### `spotify_album`

Get information about an album.

#### Arguments

Set the `album_id` to the id of the album you want to get information about in the table arguments.

The ID can be obtained by sharing an album from Spotify and copying the link. The ID is the last part of the link before the `?` character.

Example `https://open.spotify.com/intl-fr/album/3x2jF7blR6bFHtk4MccsyJ?si=u8yHWXcNTvK-VW4h9bnr3A` -> `3x2jF7blR6bFHtk4MccsyJ`

```sql
SELECT * FROM spotify_album('6jbtHi5R0jMXoliU2OS0lo');

SELECT * FROM spotify_album WHERE id = '6jbtHi5R0jMXoliU2OS0lo';
```

#### Schema

| Column index | Column name        | type    |
| ------------ | ------------------ | ------- |
| 0            | album_type         | TEXT    |
| 1            | total_tracks_album | INTEGER |
| 2            | href               | TEXT    |
| 3            | album_name         | TEXT    |
| 4            | release_date       | TEXT    |
| 5            | artist_name        | TEXT    |
| 6            | copyright          | TEXT    |
| 7            | album_popularity   | TEXT    |
| 8            | track_name         | TEXT    |
| 9            | track_duration_ms  | TEXT    |
| 10           | track_disc_number  | TEXT    |
| 11           | track_explicit     | TEXT    |
| 12           | track_href         | TEXT    |
| 13           | track_artists      | TEXT    |
| 14           | track_number       | TEXT    |

### `spotify_track`

Get information about a track.

#### Arguments

Set the `track_id` to the id of the track you want to get information about in the table arguments.

The ID can be obtained by sharing a track from Spotify and copying the link. The ID is the last part of the link before the `?` character.

```sql
SELECT * FROM spotify_track('1Je1IMUlBXcx1Fz0WE7oPT');

SELECT * FROM spotify_track WHERE id = '1Je1IMUlBXcx1Fz0WE7oPT';
```

#### Schema

| Column index | Column name        | type    |
| ------------ | ------------------ | ------- |
| 0            | album_name         | TEXT    |
| 1            | album_release_date | TEXT    |
| 2            | artist_name        | TEXT    |
| 3            | track_name         | TEXT    |
| 4            | href               | TEXT    |
| 5            | popularity         | INTEGER |
| 6            | duration_ms        | INTEGER |
| 7            | explicit           | INTEGER |
| 8            | preview_url        | TEXT    |
| 9            | track_number       | INTEGER |

### `spotify_playlist`

Get the tracks of a playlist.

#### Arguments

Set the `playlist_id` to the id of the playlist you want to get information about in the table arguments.

The ID can be obtained by sharing a playlist from Spotify and copying the link. The ID is the last part of the link before the `?` character.

```sql

SELECT * FROM spotify_playlist('37i9dQZF1DXcBWIGoYBM5M');

SELECT * FROM spotify_playlist WHERE id = `37i9dQZF1DXcBWIGoYBM5M`;
```

#### Schema

| Column index | Column name        | type    |
| ------------ | ------------------ | ------- |
| 0            | playlist_name      | TEXT    |
| 1            | playlist_followers | INTEGER |
| 2            | playlist_owner     | TEXT    |
| 3            | playlist_href      | TEXT    |
| 4            | is_public          | INTEGER |
| 5            | album_name         | TEXT    |
| 6            | album_release_date | TEXT    |
| 7            | artist_name        | TEXT    |
| 8            | track_name         | TEXT    |
| 9            | track_href         | TEXT    |
| 10           | track_popularity   | INTEGER |
| 11           | track_duration_ms  | INTEGER |
| 12           | track_explicit     | INTEGER |
| 13           | track_preview_url  | TEXT    |
| 14           | track_number       | INTEGER |

### `spotify_search`

Search for tracks, albums, playlists, and artists.

#### Arguments

Set the `query` to the search query you want to get information about in the table arguments.

You can also set the `type` to `track`, `album`, `playlist`, or `artist` to filter the search results. If you don't set the `type`, the search will return all types. You can specify multiple types by separating them with a comma.

```sql
SELECT * FROM spotify_search('Charli XCX', 'artist');

SELECT * FROM spotify_search WHERE query = 'Sabrina Carpenter' AND type = 'artist,album';
```

#### Schema

| Column index | Column name | type |
| ------------ | ----------- | ---- |
| 0            | id          | TEXT |
| 1            | name        | TEXT |
| 2            | type        | TEXT |
| 3            | href        | TEXT |
| 4            | author      | TEXT |

### `spotify_history`

Get the last 50 tracks you listened to.

```sql
SELECT * FROM spotify_history;
```

#### Schema

| Column index | Column name        | type    |
| ------------ | ------------------ | ------- |
| 0            | id                 | TEXT    |
| 1            | played_at          | TEXT    |
| 2            | played_from        | TEXT    |
| 3            | artist_name        | TEXT    |
| 4            | track_name         | TEXT    |
| 5            | album_name         | TEXT    |
| 6            | album_release_date | TEXT    |
| 7            | href               | TEXT    |
| 8            | popularity         | INTEGER |
| 9            | duration_ms        | INTEGER |
| 10           | explicit           | INTEGER |
| 11           | preview_url        | TEXT    |
| 12           | track_number       | INTEGER |

### `spotify_saved_tracks`

Get the tracks you saved.

```sql
SELECT * FROM spotify_saved_tracks;
```

#### Schema

| Column index | Column name        | type    |
| ------------ | ------------------ | ------- |
| 0            | id                 | TEXT    |
| 1            | saved_at           | TEXT    |
| 2            | artist_name        | TEXT    |
| 3            | track_name         | TEXT    |
| 4            | album_name         | TEXT    |
| 5            | album_release_date | TEXT    |
| 6            | href               | TEXT    |
| 7            | popularity         | INTEGER |
| 8            | duration_ms        | INTEGER |
| 9            | explicit           | INTEGER |
| 10           | preview_url        | TEXT    |
| 11           | track_number       | INTEGER |

## Caveats

- While using caching, the plugin is still limited by the Spotify API rate limits.
- The plugin is limited to 50 tracks for the `spotify_history` table. I could not find a way to get more tracks after spending hours on the problem. If you have a solution, please let me know.
- Caching is done for 1 hour. Therefore, fresh data might not be available immediately. You can delete the cache by running `SELECT clear_plugin_cache('spotify');` and restarting `anyquery` to get fresh data.
