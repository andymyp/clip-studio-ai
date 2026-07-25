# ClipStudio AI

ClipStudio AI is a monorepo for discovering trending videos and turning their
best moments into short-form clips. It provides a modern creator dashboard,
YouTube and Reddit discovery, authenticated backend APIs, queue foundations,
worker tooling, and local or containerized development workflows.

## Technology

- Frontend: Next.js 16, TypeScript, Tailwind CSS 4, shadcn/ui, TanStack Query,
  Axios, Zustand, React Hook Form, Zod, Sonner, and BProgress
- Backend: Go 1.26.4, Gin, GORM, PostgreSQL, Redis, Asynq, and SSE
- Worker: Python 3.14 container runtime, FastAPI, Redis, FFmpeg, yt-dlp, and
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

### Video discovery credentials

Video discovery requires an enabled provider with valid credentials. Add them
to `.env.be`. The default configuration enables YouTube and disables Reddit:

```dotenv
# YouTube Data API v3
ENABLE_YOUTUBE_API=true
YOUTUBE_API_KEY=
YOUTUBE_REGION=
YOUTUBE_LANGUAGE=en
YOUTUBE_DISCOVERY_QUERY=podcast|interview|education|business|technology|science|story|debate|speech|documentary
YOUTUBE_REUSABLE_ONLY=true
YOUTUBE_EXCLUDE_MUSIC=true
YOUTUBE_EXCLUDED_TERMS=religion,religious,faith,church,christian,muslim,islam,hindu,politics,political,election,war,weapon,gun,violence,violent,crime,murder,adult,sexual,gambling,casino,drug
YOUTUBE_DISCOVERY_DAYS=30
YOUTUBE_MIN_DURATION_SECONDS=180

# Reddit application-only OAuth
ENABLE_REDDIT_API=false
REDDIT_CLIENT_ID=
REDDIT_CLIENT_SECRET=
REDDIT_USER_AGENT=web:clipstudio-ai:v0.1.0

DISCOVERY_MIN_RESULTS=10
DISCOVERY_LIMIT=20
DISCOVERY_TIMEOUT=12s
DISCOVERY_CACHE_TTL=6h
```

Create the YouTube key in a Google Cloud project with YouTube Data API v3
enabled. Create a Reddit script application for the client ID and secret, and
use an identifiable user-agent for Reddit requests.

`ENABLE_YOUTUBE_API` and `ENABLE_REDDIT_API` control which adapters are loaded.
A disabled provider makes no authentication or API requests, even if its
credentials remain in the environment file.

The default YouTube policy is optimized for source material that can become
short-form clips:

- searches globally without restricting results to one country;
- prioritizes English-language results;
- limits discovery to videos published in the last 30 days;
- orders candidates by view count as a practical viral signal;
- requires Creative Commons (`CC BY`) videos;
- requires embeddable videos that can play outside YouTube;
- excludes YouTube's Music category and common music-title patterns;
- excludes configured sensitive topics by matching whole words in titles and
  descriptions, including religion, politics, elections, war, weapons,
  violence, crime, adult content, gambling, and drugs;
- excludes source videos shorter than three minutes;
- searches only the configured priority topics when no keyword is provided:
  podcasts, expert interviews, educational explainers, tutorials, founder and
  business advice, career advice, technology, science, personal stories, life
  lessons, debates, expert opinions, public-domain speeches, and documentaries.
  The API query uses compact umbrella terms because overly long YouTube OR
  expressions can return an empty result set.

Adjust the window and minimum source duration with
`YOUTUBE_DISCOVERY_DAYS` and `YOUTUBE_MIN_DURATION_SECONDS`. Set
`YOUTUBE_REUSABLE_ONLY=false` only when you have another rights-checking
workflow. Leave `YOUTUBE_REGION` empty for global discovery, or set an ISO
3166-1 alpha-2 country code when regional discovery is needed. Customize the
pipe-separated priority list with `YOUTUBE_DISCOVERY_QUERY`.
Customize the comma-separated sensitive-topic list with
`YOUTUBE_EXCLUDED_TERMS`. Keep the list reasonably short and use specific
terms to avoid excluding unrelated educational content.

The backend starts when an enabled provider is not configured, but that
provider is omitted from discovery. Discovery returns results from every
enabled, configured provider that responds successfully, so a temporary
failure from one provider does not discard results from the other. If no
provider is both enabled and configured, `/videos/search` returns HTTP `503`.

Normalized discovery results are cached in Redis for six hours by default.
Repeated trending searches, keyword searches, and link resolutions use the
cache without calling YouTube or Reddit again. Set `DISCOVERY_CACHE_TTL=0s` to
disable caching, or increase it to reduce provider usage further.

Discovery returns at most `DISCOVERY_LIMIT` results and targets at least
`DISCOVERY_MIN_RESULTS`. YouTube requests up to 50 candidates in the first
search call. A second page is requested only when filtering leaves fewer than
the configured minimum, keeping provider usage low while targeting 10–20
results. The minimum is best-effort because YouTube may not have ten videos
that satisfy every active safety, licensing, language, and duration filter.

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
| `/clips/trending-videos` | YouTube and Reddit discovery results     |
| `/clips/review/:externalId` | Cached clip preview and multi-select    |
| `/logs`                  | Backend clip-analysis queue status       |

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
| GET    | `/videos/search`                 | Discover trending YouTube and Reddit videos  |
| GET    | `/videos/search?keyword=podcast` | Search YouTube and Reddit videos             |
| GET    | `/videos/search?url=https://...` | Resolve one supported YouTube or Reddit link |
| POST   | `/clips/analyze`                  | Queue subtitle extraction and clip analysis   |
| GET    | `/clips/reviews/:externalId`       | Read a cached video review                     |
| GET    | `/clips/reviews/:externalId/events` | Stream review progress using SSE              |
| GET    | `/api/jobs/:id/events`           | Stream analysis progress using SSE           |
| GET    | `/api/logs`                      | List authenticated queue-job status          |

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

### Discovery behavior

Calling `/videos/search` without query parameters loads trending content:

- YouTube uses the Data API `mostPopular` chart;
- Reddit uses the authenticated `r/popular/hot` listing and keeps
  Reddit-hosted video posts.

The `keyword` and `url` parameters are mutually exclusive. Keyword searches
rank YouTube results by view count and Reddit results by top score for the
week. Link resolution accepts regular YouTube watch, Shorts, `youtu.be`, and
Reddit post links.

Both providers are normalized into `VideoSearchResult`:

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

YouTube supplies views, likes, comments, thumbnails, and duration. Reddit
supplies score, comments, thumbnail, duration, and a temporary playable media
URL. The discovery service only reads metadata and playback URLs; it does not
download source videos.

`reusable: true` means YouTube reports a Creative Commons Attribution license.
CC BY reuse requires attribution to the original creator. It is still your
responsibility to confirm the license, third-party material, privacy rights,
and platform policies before publishing. Reddit does not expose an equivalent
reuse license through this integration, so Reddit results are not marked as
reusable.

From `/clips`, **Search Trending** opens the discovery page with trending
results. **Clip By Link** resolves the submitted link on the same page. A
result can be previewed in a responsive modal—YouTube uses an embed and Reddit
uses its hosted video stream. Selecting **Clip** queues a YouTube analysis job
and opens its progress page.

### Transcript and clip pipeline

The Python worker processes a queued YouTube video in three stages:

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
every generated partial clip, and
allows multiple candidates to be selected. The **Render selected** control
records the intended selection in the UI only; rendering is deliberately not
implemented yet.

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
- normalized YouTube and Reddit video discovery without downloading media;
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
- render processing and final 9:16 composition;
- persisting worker clip candidates into PostgreSQL;
- prompt management and clip editing interfaces.

See [docs/architecture.md](docs/architecture.md) for the intended service
boundaries and data flow. See [docs/database.md](docs/database.md) for tables,
relations, constraints, indexes, and migration behavior.
