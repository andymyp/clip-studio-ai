# ClipStudio AI

ClipStudio AI is a monorepo for recommending high-potential videos and turning their
best moments into short-form clips. It provides a modern creator dashboard,
scheduled YouTube recommendation ingestion, authenticated backend APIs, queue foundations,
worker tooling, and local or containerized development workflows.

## Technology

- Frontend: Next.js 16, TypeScript, Tailwind CSS 4, shadcn/ui, TanStack Query,
  Axios, Zustand, React Hook Form, Zod, Sonner, and BProgress
- Backend: Go 1.26.4, Gin, GORM, PostgreSQL, Redis, Asynq, and SSE
- Worker: Python 3.12+, FastAPI, Redis, FFmpeg, yt-dlp, Ollama, and
  Ollama (`qwen2.5:7b`)
- Infrastructure: Docker Compose, PostgreSQL 16, Redis 7, and Ollama
- Monorepo tooling: pnpm and Turborepo

## Repository structure

```text
clip-studio-ai/
|-- apps/
|   |-- frontend/       Next.js application and Dockerfile
|   |-- backend/
|   |   |-- cmd/        API and migration entry points
|   |   |-- internal/
|   |   |   |-- api/          HTTP handlers, routing, and app wiring
|   |   |   |-- service/      Application use cases
|   |   |   |-- repository/   PostgreSQL, GORM, and data contracts
|   |   |   |-- model/        Persistent domain models
|   |   |   |-- middleware/   Gin request middleware
|   |   |   |-- queue/        Asynq tasks and client setup
|   |   |   `-- sse/          Job progress event streams
|   |   `-- Dockerfile
|   `-- worker/         Python API, background worker, and Dockerfile
|-- storage/
|   |-- input/          Source media
|   |-- output/         Generated media
|   `-- temp/           Temporary processing files
|-- docs/               Architecture documentation
|-- scripts/            Development setup scripts
|-- docker-compose.yml
|-- package.json
`-- turbo.json
```

## Prerequisites

Install the following tools:

- Node.js 24 or later
- pnpm 11
- Go 1.26.4 or later
- Python 3.12-3.14 (Docker uses Python 3.14)
- Docker Desktop with Docker Compose

Enable the pnpm version declared by the repository:

```powershell
corepack enable
corepack prepare pnpm@11.17.0 --activate
```

## Environment files

The repository contains separate templates for each configuration boundary:

| Template              | Runtime file  | Responsibility                               |
| --------------------- | ------------- | -------------------------------------------- |
| `.env.root.example`   | `.env.root`   | Compose ports and infrastructure credentials |
| `.env.fe.example`     | `.env.fe`     | Browser-safe frontend configuration          |
| `.env.be.example`     | `.env.be`     | Backend configuration                        |
| `.env.worker.example` | `.env.worker` | Worker API and job worker configuration      |

The setup script creates missing runtime files without replacing existing
files. Keep the PostgreSQL credentials in `.env.root` aligned with the
`DATABASE_URL` in `.env.be`.

To create the files manually from the repository root:

```powershell
Copy-Item .env.root.example .env.root
Copy-Item .env.fe.example .env.fe
Copy-Item .env.be.example .env.be
Copy-Item .env.worker.example .env.worker
```

These commands overwrite existing destination files only when `-Force` is
added. Review and customize the generated files before starting the services.

### Video recommendation credentials

The recommendation scheduler requires an enabled YouTube provider:

```dotenv
# YouTube Data API v3
ENABLE_YOUTUBE_API=true
YOUTUBE_API_KEY=
YOUTUBE_REGION=
YOUTUBE_RECOMMENDATION_KEYWORDS=podcast,interview,debate,speech,documentary,education,science,technology,history,business,startup,finance,psychology,motivation,health,story
RECOMMENDATION_SYNC_INTERVAL=30m
RECOMMENDATION_KEYWORDS_PER_RUN=2
DISCOVERY_LIMIT=20
DISCOVERY_TIMEOUT=12s
DISCOVERY_CACHE_TTL=6h
```

Create the YouTube key in a Google Cloud project with YouTube Data API v3
enabled. Every 30 minutes one backend instance acquires a Redis lock and
refreshes the recommendation catalog. It combines
`videos.list(chart=mostPopular)` with a rotating subset of configured topics.
With two topics per run, all 16 default topics refresh every four hours while
staying near the standard daily YouTube quota. Topic searches use `order=date`,
a 24-hour `publishedAfter` window,
`type=video`, and `videoDuration=long`.

Candidate IDs are deduplicated and enriched through `videos.list` in batches
of 50. PostgreSQL stores the detected language, topic, metrics, subscriber
snapshot, and viral score. The score weights view velocity at 35%; like ratio,
comment ratio, and freshness at 15% each; and channel growth and subscriber
ratio at 10% each. Factors are normalized as percentiles within each language
and topic cohort. Videos may belong to multiple topics.
Only videos whose YouTube metadata reports the `creativeCommon` license are
stored or returned. Older non-reusable catalog rows are removed automatically.

`GET /videos/recommendations` reads PostgreSQL and Redis only, filters by
language and topic, orders by viral score, and returns at most
`DISCOVERY_LIMIT` rows. User searches never call YouTube.
`GET /videos/recommendations/status` reports scheduler timestamps, selected
topics, partial failures, and collected video counts. Recommendations older
than seven days and channel snapshots older than 30 days are removed
automatically.

If the recommendation catalog is empty, backend startup immediately launches a
background warm-up refresh. The status endpoint remains available so the
frontend can report progress. Later restarts reuse the existing catalog and
wait for the normal scheduler interval.

## Terminal-first development

Application services run directly in local terminals. Only PostgreSQL, Redis,
and Ollama run in Docker.

### 1. Install dependencies

Run once from the repository root:

```powershell
pnpm bootstrap
```

The command:

- creates missing environment files;
- installs JavaScript dependencies;
- downloads Go modules;
- creates `apps/worker/.venv`;
- installs Python worker and development dependencies.

The worker commands call `apps/worker/.venv/Scripts/python.exe` directly, so
manual virtual-environment activation is not required.

### 2. Start development

Start PostgreSQL, Redis, and Ollama, followed by the frontend, backend, and
worker API:

```powershell
pnpm dev:all
```

Press `Ctrl+C` to stop the attached application processes. Stop the
infrastructure containers separately:

The background worker handles `SIGINT` and `SIGTERM` gracefully: it stops
claiming jobs, returns any unstarted claimed job to Redis, finishes the active
job, acknowledges it, and closes both Redis connections. Docker allows up to
15 minutes for an active FFmpeg render to finish before force-stopping the job
container. Interrupted queue entries are recovered automatically at startup.

```powershell
pnpm infra:down
```

### Individual processes

Run these commands in separate terminals when you want independent logs:

```powershell
pnpm infra:up
pnpm dev:frontend
pnpm dev:backend
pnpm dev:worker-api
```

Run the asynchronous job process to extract transcripts, rank moments, and
create partial clip files:

```powershell
pnpm dev:worker-jobs
```

Other combined modes:

| Command         | Services started                                          |
| --------------- | --------------------------------------------------------- |
| `pnpm dev`      | Frontend, backend, worker API, and background job process |
| `pnpm dev:all`  | Infrastructure followed by `pnpm dev`                     |
| `pnpm dev:full` | Alias for `pnpm dev`                                      |

`pnpm dev:full` does not start infrastructure automatically. Run
`pnpm infra:up` first.

## Local endpoints

| Service        | URL                          |
| -------------- | ---------------------------- |
| Frontend       | http://localhost:3000        |
| Backend health | http://localhost:3001/health |
| Worker API     | http://localhost:3002        |
| Worker health  | http://localhost:3002/health |
| PostgreSQL     | localhost:5432               |
| Redis          | localhost:6379               |
| Ollama         | http://localhost:11434       |

Ollama models are not downloaded automatically. Pull the configured model when
it is first required:

```powershell
docker compose --env-file .env.root exec ollama ollama pull qwen2.5:7b
```

YouTube extraction uses Node 22 or newer, Chrome request impersonation, request
pacing, exponential retries, and one dedicated Netscape-format cookie file.
Terminal development reads the repository-local secret:

```dotenv
YTDLP_COOKIE_FILE=../../storage/secrets/youtube-cookies.txt
```

Export a dedicated YouTube session to
`storage/secrets/youtube-cookies.txt`. Follow yt-dlp's private-session export
procedure: sign in from a private browser window, navigate in the same tab to
`https://www.youtube.com/robots.txt`, export only YouTube cookies in Netscape
format with a trusted local exporter, then close that private session
permanently.

Compose mounts the same host file read-only as a Docker secret and changes only
its in-container path:

```dotenv
YTDLP_COOKIE_FILE=/run/secrets/youtube_cookies
```

The host location can be changed in `.env.root`:

```dotenv
YTDLP_COOKIE_FILE_HOST=./storage/secrets/youtube-cookies.txt
```

Cookie files and `storage/secrets/*` are excluded from Git and Docker build
contexts. Never commit, bake into an image, log, or share the cookie file.

### Frontend routes

| Route                    | Purpose                                  |
| ------------------------ | ---------------------------------------- |
| `/signin`                | JWT login                                |
| `/signup`                | Account registration                     |
| `/dashboard`             | Workspace overview and suggested sources |
| `/clips`                 | Searchable generated-clips library       |
| `/clips/recommendations` | Database-backed YouTube recommendations |
| `/clips/review/:externalId` | Cached clip preview and multi-select    |
| `/logs`                  | Persisted render jobs, progress, and retry |

Dashboard routes are client-protected and use Axios JWT refresh interceptors.
TanStack Query manages discovery results and backend queue logs, while Zustand
persists only the authenticated session.

### Backend API

| Method | Endpoint                         | Purpose                                      |
| ------ | -------------------------------- | -------------------------------------------- |
| GET    | `/health`                        | Service health                               |
| POST   | `/auth/register`                 | Register and receive a token pair            |
| POST   | `/auth/login`                    | Authenticate and receive tokens              |
| POST   | `/auth/refresh`                  | Rotate a valid refresh token                 |
| GET    | `/videos/recommendations?language=en&keywords=podcast` | Query cached recommendations |
| GET    | `/videos/search?url=https://...` | Resolve one supported YouTube link |
| POST   | `/clips/analyze`                  | Queue subtitle extraction and clip analysis   |
| GET    | `/clips/reviews/:externalId`       | Read a cached video review                     |
| GET    | `/clips/reviews/:externalId/events` | Stream review progress using SSE              |
| GET    | `/clips/rendered`                 | Paginated rendered clips library              |
| GET    | `/renders`                       | Paginated render jobs                         |
| GET    | `/renders/events`                | Stream render-job changes using SSE           |
| GET    | `/api/jobs/:id/events`           | Stream analysis progress using SSE           |
| GET    | `/api/logs`                      | List authenticated queue-job status          |

`/clips/rendered` accepts `page`, `page_size`, `search`, and
`sort=newest|oldest|score`. `/renders` accepts `page`, `page_size`,
`status=queued|pending|processing|completed|failed|cancelled`, and
`sort=newest|oldest|progress`. Filtering, ordering, counting, and pagination
are applied by PostgreSQL; the frontend renders the returned page as-is.

All `/videos/*`, `/clips/*`, and `/api/*` routes require an access token:

```text
Authorization: Bearer <access_token>
```

Register and login accept an email and a password between 8 and 72 bytes:

```json
{
  "email": "user@example.com",
  "password": "a-strong-password"
}
```

Passwords are stored only as bcrypt hashes. Access tokens expire after 15
minutes by default. Refresh tokens expire after seven days, are backed by
Redis, and rotate on every successful refresh; replaying a consumed refresh
token is rejected.

### Recommendation behavior

`/videos/recommendations?language=en&keywords=podcast` reads the persisted
recommendation catalog, orders it by `viral_score DESC`, and caches the result.
The endpoint never contacts YouTube. `/videos/search?url=...` remains available
only for resolving a direct YouTube watch, Shorts, or `youtu.be` link.

Recommendations are returned as `VideoSearchResult`:

```json
{
  "external_id": "provider-video-id",
  "platform": "youtube",
  "category_id": "27",
  "title": "Video title",
  "url": "https://www.youtube.com/watch?v=...",
  "embed_url": "https://www.youtube.com/embed/...",
  "media_url": "",
  "thumbnail": "https://...",
  "duration": 154,
  "views": 1200000,
  "likes": 42000,
  "comments": 1800,
  "score": 0,
  "license": "creativeCommon",
  "reusable": true
}
```

YouTube supplies views, likes, comments, thumbnails, duration, language,
publication time, and channel subscriber snapshots. The scheduler only reads
metadata; it does not download source videos.

`reusable: true` means YouTube reports a Creative Commons Attribution license.
CC BY reuse requires attribution to the original creator. It is still your
responsibility to confirm the license, third-party material, privacy rights,
and platform policies before publishing.

From `/clips`, **Search Recommendation** opens the database-backed recommendation
page. **Clip By Link** resolves the submitted link on the same page. A
result can be previewed in a responsive modal with a YouTube embed. Selecting
**Clip** queues a YouTube analysis job
and opens its progress page.

### Transcript and clip pipeline

The Python worker first analyzes a queued YouTube video:

1. `TranscriptService` invokes yt-dlp with `--skip-download`,
   `--write-auto-subs`, and `--write-subs`. It requests English VTT subtitles
   without downloading video media, then converts VTT cues into JSON:

   ```json
   [
     {
       "text": "A useful standalone thought.",
       "start": 120,
       "end": 125
     }
   ]
   ```

2. `ClipRankingService` sends the transcript JSON to Ollama using
   `qwen2.5:7b`. Candidates are scored for hook strength, emotion, curiosity,
   replay probability, and information density. Only complete moments between
   20 and 60 seconds are accepted.
3. `PartialDownloaderService` invokes yt-dlp separately for each accepted
   range using `--download-sections`, the FFmpeg downloader, and
   `--force-keyframes-at-cuts`. It never requests a full-video download.

Job state and results are stored in Redis for seven days. Generated transcript
JSON is written under `storage/output/transcripts/`, and partial MP4 files are
written under `storage/output/clips/<job-id>/`.

Completed reviews are cached per user and source-video `external_id` for 30
days. Selecting **Clip** again reuses that successful result and its generated
media. Failed analysis jobs are not cached; their transcript JSON and entire
per-job clip directory are deleted before the failed state is published.

The review page receives job updates over an authenticated SSE stream, previews
every generated partial clip, and allows multiple candidates to be selected.
The **Render selected** control opens the render options dialog.
Submitting selected candidates creates one PostgreSQL render job per clip. The
worker generates word-timed ASS subtitles, burns them into a 9:16 H.264/AAC
video with watermark and source attribution, then asks Ollama for a title,
description, and hashtags. Progress is persisted through the backend, failed
jobs can be retried from `/logs`, and completed exports appear on `/clips`.

Each render produces one best-potential master cut. Before encoding, the worker
detects and removes dead air and filler-only segments, samples the dominant face
for speaker-aware vertical reframing, inserts subtle visual beats, generates a
truthful opening hook, and emphasizes important phrases in the ASS captions.
Packaging first extracts a structured main-context brief containing the topic,
central claim, and payoff from the full selected transcript and source title.
It then produces three title candidates and three independent hook candidates
and scores their relevance to that brief before editorial validation. Publishing
titles must be accurate, concise, payoff-led, and front-loaded; opening hooks
must be 3-6 words, distinct from both the title and dialogue, and free of
Markdown, generic clickbait, fragments, repeated words, or excessive capitals.
Invalid AI output is replaced with separate deterministic title and hook
fallbacks rather than copied transcript text.
Review sections request the best compatible source up to 1080p. Final exports
use Lanczos scaling, eased subject-following pans, short safe-area captions,
larger boxed branding, the discovered YouTube handle, and high-quality H.264
CRF 17 / AAC 192 kbps encoding.

The upgraded pipeline snaps AI candidates to transcript sentence boundaries,
uses sentence endings for visual beats, samples faces at three points per shot,
locks small camera movements, and limits pan velocity. The opening uses a subtle
push-in and fade while keeping the hook card brief. Caption chunks break on
clauses, avoid orphan words, and allocate at least 700 ms when the timeline
permits. Exact duplicate cues, progressive subtitle overlap, and cues spanning
an edit boundary are collapsed before ASS generation.

Every render uses Smart AI Auto Crop; there is no manual render-profile choice.
The reframe planner samples each semantic shot, identifies the most active face
from face size and mouth-region activity, falls back to object contours and
frame-motion action detection, and produces a full-screen 9:16 crop. Pan and
zoom are eased, velocity-limited, and stabilized with dead zones so the camera
tracks meaningful movement rather than floating.

Audio is mastered in two passes. FFmpeg first measures loudness and then applies
an 80 Hz high-pass filter, light denoising, voice compression, EBU R128
normalization to -16 LUFS / -1.5 dBTP, and a true-peak limiter. After encoding,
the worker runs an FFprobe/FFmpeg quality gate that verifies 1080x1920 output,
audio presence, planned duration, audio/video drift, black frames, frozen
frames, and complete decodability. Failed validation marks the render failed
instead of publishing a broken file.
Ranked moments include a two-second context lead-in and a 1.2-second context
tail. The renderer preserves those frames, mutes audio in both edge buffers,
shows a large, outlined, subtitle-style hook in the center without a background
during the opening, suppresses closing
captions, and uses a single-line `Source: YT @handle` badge. Caption sanitation
keeps letters, numbers, whitespace, and standard punctuation while removing
emoji, currency marks, arrows, and other symbols.
Dialogue above the readability threshold is adaptively slowed toward three
words per second, with a conservative 0.85x limit. Video, audio, captions,
opening hook, and silent edge buffers are retimed together so synchronization
is preserved; naturally paced dialogue remains at its original speed.
Published view, engagement, watch-time, and completion metrics can be recorded
from the clip preview. These produce a viral score and teach later analyses the
duration range that has performed best.
Performance feedback also accepts engaged views, swipe-away percentage,
replays, and the primary drop-off timestamp. These signals influence learned
duration weighting. Source license and reusable status are retained from
discovery, and the render dialog requires explicit confirmation that the user
has permission to reuse the content.

The backend and background worker use two-stage shutdown. The first `SIGINT` or
`SIGTERM` stops new intake and drains active HTTP requests or the current media
job. A second signal force-closes the backend; the worker interrupts the active
job and returns it to Redis so it can be recovered on the next start. Both
processes close database and Redis connections before exiting. On Windows the
job worker also handles `SIGBREAK`. Docker grants render workers up to 15
minutes to finish an active FFmpeg render.

SSE messages use the job status as the event name and send progress as JSON:

```text
event:queued
data:{"step":"queued","percentage":0}
```

The backend handles `SIGINT` and `SIGTERM` gracefully. It stops accepting new
requests, drains active requests for `SHUTDOWN_TIMEOUT` (10 seconds by
default), then closes its Asynq, Redis, and PostgreSQL connections. Docker
Compose allows a 15-second grace period before forcing the container to stop.

## Full Docker stack

After creating the environment files with `pnpm bootstrap`, build and start
all seven services:

```powershell
pnpm docker:up
```

Equivalent Docker Compose command:

```powershell
docker compose --env-file .env.root up --build -d
```

The Compose stack provides:

- health-gated dependency startup;
- a dedicated `clipstudio-network` bridge network;
- persistent PostgreSQL, Redis, and Ollama volumes;
- shared local media storage under `storage/`;
- configurable host ports;
- service-specific environment files.

### Docker commands

| Command                    | Purpose                                       |
| -------------------------- | --------------------------------------------- |
| `pnpm docker:up`           | Build and start the complete stack            |
| `pnpm docker:build`        | Build application images                      |
| `pnpm docker:logs`         | Follow logs from all services                 |
| `pnpm docker:ps`           | Show service status and health                |
| `pnpm docker:down`         | Stop and remove containers                    |
| `pnpm docker:down:volumes` | Remove containers and persistent data volumes |
| `pnpm db:migrate`          | Apply additive GORM database migrations       |

Warning: `pnpm docker:down:volumes` permanently removes local PostgreSQL,
Redis, and Ollama volume data.

## Quality commands

| Command          | Purpose                                 |
| ---------------- | --------------------------------------- |
| `pnpm build`     | Build Turborepo packages                |
| `pnpm lint`      | Run frontend linting                    |
| `pnpm typecheck` | Run TypeScript checks                   |
| `pnpm test`      | Run Go and Python tests                 |
| `pnpm check`     | Run linting, type checks, and all tests |
| `pnpm format`    | Format supported repository files       |

Python-related package scripts use `apps/worker/.venv` automatically.

## Current scope

Included:

- application shells and process entry points;
- configuration loading;
- infrastructure clients;
- health endpoints;
- scheduled, database-backed YouTube recommendations without downloading media;
- YouTube VTT subtitle extraction without downloading source media;
- VTT-to-JSON transcript conversion;
- Ollama clip ranking with `qwen2.5:7b`;
- section-only MP4 downloads for ranked moments;
- generated-clip preview and multi-selection;
- queued analysis jobs and SSE progress events;
- JWT registration, login, rotating refresh tokens, and Bearer middleware;
- responsive sign-in, sign-up, dashboard, clips, discovery, preview, and
  queue-log interfaces;
- container builds and local orchestration;
- persistent development storage.

Deferred:

- uploads and media ingestion;
- a timeline editor and manual subtitle corrections;
- direct publishing integrations for supported social platforms;
- prompt management and clip editing interfaces.

See [docs/architecture.md](docs/architecture.md) for the intended service
boundaries and data flow. See [docs/database.md](docs/database.md) for tables,
relations, constraints, indexes, and migration behavior.
