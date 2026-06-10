# Atlas Cloud plugin

Query [Atlas Cloud](https://www.atlascloud.ai/?utm_source=github&utm_medium=link&utm_campaign=anyquery) with SQL. Atlas Cloud is a full-modal AI inference platform that gives developers a single AI API to access video generation, image generation, and LLM APIs — 300+ curated models across all modalities.

The plugin exposes four tables:

| Table                   | What it does                                                            |
| ----------------------- | ----------------------------------------------------------------------- |
| `atlascloud_models`     | The model catalog (read-only, cached for 6 hours)                       |
| `atlascloud_llm`        | Synchronous chat completion (one row per call)                          |
| `atlascloud_image`      | Image generation (blocks until the image is ready, 90 s at most)        |
| `atlascloud_video_jobs` | Async video jobs: `INSERT` to submit, `SELECT` to poll, `DELETE` to clean up |

## Installation

```bash
anyquery install atlascloud
```

You will be asked for:

- `api_key` (**required**): your Atlas Cloud API key. Create one in the [Atlas Cloud console](https://www.atlascloud.ai) under Settings → API Keys.
- `base_url` (optional): an override of the API endpoint. Leave empty unless you know what you are doing.

> ⚠️ The `llm`, `image` and `video_jobs` tables make **paid** API calls. A join can fan out into one call per row — always test your query with a `LIMIT` first.

## `atlascloud_models`

Lists every model available on Atlas Cloud, with the exact `id` to pass to the other tables.

| Column        | Type | Description                                                            |
| ------------- | ---- | ---------------------------------------------------------------------- |
| `id`          | TEXT | The model identifier (e.g. `deepseek-ai/DeepSeek-V3-0324`)              |
| `name`        | TEXT | Display name                                                           |
| `modality`    | TEXT | `llm`, `image`, `video` or `audio`                                     |
| `provider`    | TEXT | Upstream provider (e.g. `BYTEDANCE`, `GOOGLE`)                         |
| `description` | TEXT | Short description. Might be NULL                                       |
| `price`       | TEXT | JSON object with the pricing (per-token for LLMs, per-generation base price for image/video). Might be NULL |

```sql
-- List the available video models
SELECT id, name, provider FROM atlascloud_models WHERE modality = 'video';
```

The catalog is cached on disk for 6 hours. Force a refresh with `SELECT clear_plugin_cache('atlascloud');`.

## `atlascloud_llm`

Runs a chat completion and returns one row per call.

Parameters (passed as `WHERE` clauses or with `atlascloud_llm(model, prompt, system_prompt, temperature, max_tokens)`):

| Parameter       | Type    | Required | Description                       |
| --------------- | ------- | -------- | --------------------------------- |
| `model`         | TEXT    | yes      | e.g. `deepseek-ai/DeepSeek-V3-0324` |
| `prompt`        | TEXT    | yes      | The user message                  |
| `system_prompt` | TEXT    | no       | An optional system message        |
| `temperature`   | REAL    | no       | 0.0–2.0                           |
| `max_tokens`    | INTEGER | no       |                                   |

Output columns: `response`, `finish_reason`, `prompt_tokens`, `completion_tokens`.

```sql
-- A single completion
SELECT response FROM atlascloud_llm('qwen/qwen3-8b', 'Explain SQL in one sentence');

-- Per-row LLM calls via a join: summarize your GitHub issues
SELECT title, response AS summary
FROM github_my_issues, atlascloud_llm
WHERE atlascloud_llm.model = 'deepseek-ai/DeepSeek-V3-0324'
  AND atlascloud_llm.prompt = 'Summarize in one sentence: ' || body
LIMIT 10;
```

## `atlascloud_image`

Generates an image and blocks until it is ready (polls every 2 seconds, gives up after 90 seconds — typical generations take 2–10 s).

Parameters: `model` (required), `prompt` (required), `image_url` (optional, for image-to-image models), `extra_params` (optional JSON object of model-specific parameters, e.g. `'{"image_size": "1024x1024"}'`).

| Column          | Type | Description                                          |
| --------------- | ---- | ---------------------------------------------------- |
| `url`           | TEXT | The URL of the generated image; NULL if it failed    |
| `prediction_id` | TEXT | The Atlas Cloud prediction ID                        |
| `status`        | TEXT | `completed`, `failed` or `timeout`                   |
| `error`         | TEXT | The error message on failure, NULL otherwise         |

```sql
SELECT url, status, error
FROM atlascloud_image('bytedance/seedream-3.0/text-to-image', 'A serene Japanese garden, watercolor style');
```

A failed generation returns a row with `status`/`error` set instead of failing the query, so batch generation over a join continues past individual failures. Unsupported model parameters come back as a `failed` status with the API's error message.

## `atlascloud_video_jobs`

Video generation takes 30 seconds to 3 minutes, so it is exposed as an async job table: `INSERT` submits a job and returns immediately; `SELECT` polls the pending jobs once and returns their current state. Jobs are persisted locally (per profile), so they survive across anyquery sessions.

INSERT columns: `model` (required), `prompt` (required), `image_url` (optional, for image-to-video models), `extra_params` (optional JSON object, e.g. `'{"duration": 5}'`).

SELECT columns:

| Column          | Type | Description                                                   |
| --------------- | ---- | ------------------------------------------------------------- |
| `prediction_id` | TEXT | The Atlas Cloud prediction ID (primary key)                   |
| `model`         | TEXT | As submitted                                                  |
| `prompt`        | TEXT | As submitted                                                  |
| `status`        | TEXT | `processing`, `completed` or `failed`                         |
| `outputs`       | TEXT | JSON array of output URLs; NULL while processing              |
| `error`         | TEXT | NULL unless the job failed                                    |
| `created_at`    | TEXT | When the job was submitted (RFC 3339)                         |

```sql
-- Submit a job
INSERT INTO atlascloud_video_jobs(model, prompt)
VALUES ('bytedance/seedance-2.0/text-to-video', 'Ocean waves at sunset');

-- … wait 30 s to 3 min, then poll
SELECT prediction_id, status, outputs FROM atlascloud_video_jobs;

-- Clean up finished jobs (removes them from the local store only)
DELETE FROM atlascloud_video_jobs WHERE status = 'completed';
```

Jobs already in a terminal state are returned from local storage without re-polling. The store keeps the 1000 most recent jobs per profile.

## Caveats

- **Output URLs may expire.** The image/video URLs are hosted by Atlas Cloud and may be temporary — download your outputs promptly.
- **Duplicate-call protection.** SQLite can re-scan a table during a query (joins, OR clauses). To avoid double-billing, `llm` and `image` results are memoized in memory for 5 minutes, keyed on the full input tuple. Within that window, the exact same call returns the same generation. Restarting anyquery clears it; nothing is persisted.
- **Generations are never cached on disk.** Only the model catalog is cached. `SELECT clear_plugin_cache('atlascloud')` clears the catalog cache **and the local video job list**.
- **Server mode timeouts.** `atlascloud_image` can block up to 90 s inside a query. Some MySQL clients default to shorter read timeouts — raise your client's read timeout if needed.
- **Rate limits and balance.** HTTP 429 responses are retried with backoff (honoring `Retry-After`). "Insufficient balance" errors are surfaced as-is: top up in the [Atlas Cloud console](https://www.atlascloud.ai/console/billing).
- **Model-specific parameters** (resolution, duration, seed…) vary per model and are passed through `extra_params` untouched. Check each model's page on Atlas Cloud for its schema.
