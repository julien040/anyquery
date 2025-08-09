# Readwise

List and upsert/delete highlights and documents from Readwise and Readwise Reader

## Installation

First, install the plugin:

```bash
anyquery plugin install readwise
```

Anyquery will ask you to create a Readwise API token. To create one, go to [https://readwise.io/access_token](https://readwise.io/access_token) and click on "Get Access Token". Copy the token and paste it in the plugin configuration.

## Usage

### Highlights

The plugin lets you CRUD highlights from Readwise.

```sql
-- List all highlights
SELECT * FROM readwise_highlights;

-- Insert a new highlight
INSERT INTO readwise_highlights (text, note, book_source, book_title) VALUES ('Lorem ipsum dolor sit amet.', 'My annotation note.', 'https://source.example.com/highlight/abc', 'My Book Title');

-- Update an existing highlight
UPDATE readwise_highlights SET text = 'Updated text.' WHERE text = 'Lorem ipsum dolor sit amet.';

-- Delete an existing highlight
DELETE FROM readwise_highlights WHERE text = 'Updated text.';
```

### Documents

The plugin lets you CRUD documents from Readwise Reader.

```sql
-- List all documents
SELECT * FROM readwise_documents;

-- Insert a new document
INSERT INTO readwise_documents (source_url, title, author, category) VALUES ('https://source.example.com/article/abc', 'My Article Title', 'John Doe', 'archive');

-- Update an existing document
UPDATE readwise_documents SET title = 'Updated title.' WHERE title = 'My Article Title';

-- Delete an existing document
DELETE FROM readwise_documents WHERE title = 'Updated title.';
```

## Schema

### readwise_highlights

| Column index | Column name          | Type     |
| ------------ | -------------------- | -------- |
| 0            | id                   | INTEGER  |
| 1            | text                 | TEXT     |
| 2            | note                 | TEXT     |
| 3            | location             | INTEGER  |
| 4            | location_type        | TEXT     |
| 5            | color                | TEXT     |
| 6            | highlighted_at       | DATETIME |
| 7            | created_at           | DATETIME |
| 8            | updated_at           | DATETIME |
| 9            | url                  | TEXT     |
| 10           | is_favorite          | BOOLEAN  |
| 11           | is_discard           | BOOLEAN  |
| 12           | tags                 | TEXT     |
| 13           | book_id              | INTEGER  |
| 14           | book_title           | TEXT     |
| 15           | book_author          | TEXT     |
| 16           | book_source          | TEXT     |
| 17           | book_category        | TEXT     |
| 18           | book_cover_image_url | TEXT     |
| 19           | book_summary         | TEXT     |

### readwise_documents

| Column index | Column name      | Type     |
| ------------ | ---------------- | -------- |
| 0            | id               | TEXT     |
| 1            | url              | TEXT     |
| 2            | source_url       | TEXT     |
| 3            | title            | TEXT     |
| 4            | author           | TEXT     |
| 5            | source           | TEXT     |
| 6            | category         | TEXT     |
| 7            | location         | TEXT     |
| 8            | tags             | TEXT     |
| 9            | site_name        | TEXT     |
| 10           | word_count       | INTEGER  |
| 11           | created_at       | DATETIME |
| 12           | updated_at       | DATETIME |
| 13           | published_date   | DATE     |
| 14           | notes            | TEXT     |
| 15           | summary          | TEXT     |
| 16           | image_url        | TEXT     |
| 17           | parent_id        | TEXT     |
| 18           | reading_progress | REAL     |
| 19           | first_opened_at  | DATETIME |
| 20           | last_opened_at   | DATETIME |
| 21           | saved_at         | DATETIME |
| 22           | last_moved_at    | DATETIME |

## Additional Information

### Rate Limits

The plugin is subject to the Readwise API Rate Limits. Essentially, you can:

- read up to 24 000 highlights per minute
- create up to 12 000 highlights per minute
- update up to 240 highlights per minute
- delete up to 240 highlights per minute
- read up to 2000 documents per minute
- create up to 50 documents per minute
- update up to 50 documents per minute
- delete up to 20 documents per minute

### Cache

The plugin uses a cache to store the highlights and documents. Documents and highlights are cached for 4 hours. Any insert/update/delete operation will invalidate the cache.
