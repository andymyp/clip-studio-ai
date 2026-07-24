# ClipStudio AI

ClipStudio AI is a monorepo foundation for an AI-assisted video clipping
platform. It provides service boundaries, dependency wiring, containers,
backend video and analysis-job APIs, and development tooling.

## Technology

- Frontend: Next.js 16, TypeScript, Tailwind CSS 4, shadcn/ui, TanStack Query,
  and Zustand
- Backend: Go 1.26.4, Gin, GORM, PostgreSQL, Redis, Asynq, and SSE
- Worker: Python 3.14 container runtime, FastAPI, FFmpeg, yt-dlp, and Faster Whisper
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

The asynchronous job process is optional at this foundation stage:

```powershell
pnpm dev:worker-jobs
```

Other combined modes:

| Command         | Services started                                          |
| --------------- | --------------------------------------------------------- |
| `pnpm dev`      | Frontend, backend, and worker API                         |
| `pnpm dev:all`  | Infrastructure followed by `pnpm dev`                     |
| `pnpm dev:full` | Frontend, backend, worker API, and background job process |

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
it is first required.

### Backend API

| Method | Endpoint                             | Purpose                              |
| ------ | ------------------------------------ | ------------------------------------ |
| GET    | `/health`                            | Service health                       |
| GET    | `/api/videos/search?keyword=podcast` | Search persisted videos              |
| GET    | `/api/videos/:id`                    | Get video metadata                   |
| POST   | `/api/videos/:id/analyze`            | Create and enqueue an analysis job   |
| GET    | `/api/jobs/:id/events`               | Stream analysis progress using SSE   |

Analysis requests return HTTP `202 Accepted`. The queued task type is
`video:analyze`; its JSON payload contains `job_id` and `video_id`. Run
`pnpm dev:worker-jobs` alongside the API when a task consumer is available.

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

After creating the environment files with `pnpm bootstrap`, build and start all six
services:

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
- persisted video search and detail endpoints;
- queued analysis jobs and SSE progress events;
- container builds and local orchestration;
- persistent development storage.

Deferred:

- authentication and authorization;
- uploads and media ingestion;
- analysis task processing;
- transcription and AI pipelines;
- prompts and product user interfaces.

See [docs/architecture.md](docs/architecture.md) for the intended service
boundaries and data flow. See [docs/database.md](docs/database.md) for tables,
relations, constraints, indexes, and migration behavior.
