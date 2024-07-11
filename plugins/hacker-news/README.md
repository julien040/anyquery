# Hacker News

This plugin allows you to do queries on Hacker News.

## Installation

```bash
anyquery install hn
```

## Usage

```bash
anyquery -q "SELECT * FROM hn_search('julien040')"
```

## Tables

### `hn_search`

Do a search using the [Algolia API](https://hn.algolia.com/api).

#### Parameters

You can specify your search query as a table argument or for the "query" column in the WHERE clause.
You can filter by type (story, comment, job, poll, pollopt) and by author. The API can retrieve up to 1000 results.

```sql
SELECT * FROM hn_search('Gut cli') -- is equivalent to
SELECT * FROM hn_search WHERE query = 'Gut cli'

SELECT * FROM hn_search('Gut cli') WHERE type = 'comment' -- Only search for comment
```

#### Schema

| Column index | Column name  | type    |
| ------------ | ------------ | ------- |
| 0            | id           | TEXT    |
| 1            | title        | TEXT    |
| 2            | created_at   | TEXT    |
| 3            | type         | TEXT    |
| 4            | url          | TEXT    |
| 5            | author       | TEXT    |
| 6            | points       | INTEGER |
| 7            | num_comments | INTEGER |
| 8            | story_id     | INTEGER |
| 9            | story_title  | TEXT    |
| 10           | comment_text | TEXT    |
| 11           | parent_id    | INTEGER |
| 12           | tags         | TEXT    |

### `hn_post`

Retrive a post by its id. The post can be a comment, a story, a job, a poll or a pollopt.

#### Parameters

You can specify the post id as a table argument or for the "id" column in the WHERE clause.

```sql
SELECT * FROM hn_post(36391655) -- is equivalent to
SELECT * FROM hn_post WHERE id = 36391655
```

#### Schema

| Column index | Column name | type    |
| ------------ | ----------- | ------- |
| 0            | by          | TEXT    |
| 1            | created_at  | TEXT    |
| 2            | title       | TEXT    |
| 3            | url         | TEXT    |
| 4            | text        | TEXT    |
| 5            | descendants | INTEGER |
| 6            | score       | INTEGER |
| 7            | type        | TEXT    |
| 8            | deleted     | INTEGER |
| 9            | dead        | INTEGER |
| 10           | parent      | INTEGER |
| 11           | poll        | INTEGER |
| 12           | kids        | TEXT    |

### `hn_descendants`

Find all the comments recursively for a given post id. The post can be a comment or a story.

#### Parameters

You can specify the post id as a table argument or for the "id" column in the WHERE clause.

```sql
SELECT * FROM hn_descendants(36391655) -- is equivalent to
SELECT * FROM hn_descendants WHERE id = 36391655
```

#### Schema

| Column index | Column name | type    |
| ------------ | ----------- | ------- |
| 0            | id          | INTEGER |
| 1            | by          | TEXT    |
| 2            | created_at  | TEXT    |
| 3            | url         | TEXT    |
| 4            | text        | TEXT    |
| 5            | type        | TEXT    |
| 6            | deleted     | INTEGER |
| 7            | dead        | INTEGER |
| 8            | parent      | INTEGER |
| 9            | kids        | TEXT    |

### `hn_user_posts`

Find the last 100 posts of a user.

#### Parameters

You can specify the user name as a table argument or for the "user" column in the WHERE clause.

```sql
SELECT * FROM hn_user_posts('julien040') -- is equivalent to
SELECT * FROM hn_user_posts WHERE id = 'julien040'
```

#### Schema

| Column index | Column name | type    |
| ------------ | ----------- | ------- |
| 0            | created_at  | TEXT    |
| 1            | title       | TEXT    |
| 2            | url         | TEXT    |
| 3            | text        | TEXT    |
| 4            | descendants | INTEGER |
| 5            | score       | INTEGER |
| 6            | type        | TEXT    |
| 7            | deleted     | INTEGER |
| 8            | dead        | INTEGER |
| 9            | parent      | INTEGER |
| 10           | poll        | INTEGER |
| 11           | kids        | TEXT    |

## Caveats

- The search API can retrieve up to 1000 results.
- The search API is rate limited to 10 000 requests per hour. It might not be suited for mass export.
- The plugin doesn't have any caching mechanism. It will make a request to the API for each query. This might be inconvenient on slow networks.
