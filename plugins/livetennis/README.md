# Live Tennis API plugin

Read tennis match snapshots with SQL:

```sql
SELECT * FROM livetennis_matches;
```

The table contains one page of up to 200 matches in progress. Queries refresh the snapshot at most once every 15 minutes.
Read `fetched_at` to check the age of a row. This plugin uses only the free live match endpoint.

## Installation

Install the plugin with Anyquery:

```sh
anyquery install livetennis
```

When Anyquery asks for `api_key`, enter your [free API key](https://livetennisapi.com/subscribe/free).
The [authentication documentation](https://docs.livetennisapi.com/auth-quota-and-health.html) describes the key and quota.

## Query matches

```sql
SELECT id, tournament, scheduled_at, fetched_at
FROM livetennis_matches
WHERE tour = 'wta';

SELECT tournament, count(*) AS match_count
FROM livetennis_matches
GROUP BY tournament;

SELECT id, json_extract(score, '$.games') AS games
FROM livetennis_matches;
```

| Column | Type | Contents |
| ------ | ---- | -------- |
| `id` | INTEGER | Match identifier. |
| `tournament` | TEXT | Tournament name. |
| `tour` | TEXT | Tour name, or SQL NULL when absent. |
| `scheduled_at` | DATETIME | Scheduled start as an RFC3339 timestamp, or SQL NULL when absent. |
| `status` | TEXT | Match lifecycle status. |
| `players` | TEXT | Player data as a JSON object with `p1` and `p2` entries. |
| `score` | TEXT | Score data as JSON, or SQL NULL when absent. |
| `fetched_at` | DATETIME | Snapshot fetch time as an RFC3339 timestamp. |

Score arrays retain the API's player order. In `score.games`, the first array belongs to player 1 and the second to player 2.
The [live score documentation](https://docs.livetennisapi.com/live-scores.html) describes these fields.

## Snapshot limits

Each fetch reads `/matches?status=live&limit=200&offset=0`. SQL filters apply to one page of up to 200 rows.

The free API tier allows 100 requests a day. A fixed 900 second floor permits at most 96 attempts in 24 hours.
Failed requests also reserve that interval. The plugin does not retry them immediately.

Profiles with identical API credentials use a shared cached snapshot and request floor across plugin processes and restarts.
Cache files live under `anyquery/plugins/livetennis` in the user cache directory, separated by a hash of the key.
Other machines and API clients share the key's daily quota.

## Local development

Build the plugin from this directory. Copy the development manifest before adding your API key:

```sh
make
cp devManifest.example.json devManifest.local.json
```

Set `api_key` in `devManifest.local.json` to your key. Start Anyquery in this directory:

```sh
anyquery --dev
```

Then load the local plugin and query it:

```sql
SELECT load_dev_plugin('livetennis', 'devManifest.local.json');
SELECT * FROM livetennis_matches;
```

Run the plugin tests with `make test`. Build the six platform binaries with `make release` and GoReleaser v2.
