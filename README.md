# ClipStudio AI

Production-oriented foundation for an AI-assisted video clipping platform. This repository is intentionally limited to infrastructure, dependency wiring, health endpoints, and service boundaries; business logic is not implemented.

## Stack

- **Frontend:** Next.js 15, TypeScript, Tailwind CSS, shadcn/ui primitives, TanStack React Query, Zustand
- **Backend:** Go 1.23, Gin, GORM, PostgreSQL, Redis, Asynq, Server-Sent Events readiness
- **Worker:** Python 3.12, FastAPI, Redis-backed Celery-style worker pattern, FFmpeg, yt-dlp, Faster Whisper
- **Infrastructure:** Docker Compose, PostgreSQL 16, Redis 7, Ollama

## Repository layout

```text
.
├── frontend/       Next.js application
├── backend/        Go HTTP API and async job producer
├── worker/         Python worker API and background worker
├── docker/         Service Dockerfiles
├── storage/        Local media/output mount points
├── docs/           Architecture documentation
├── docker-compose.yml
└── turbo.json
```

## Quick start

1. Copy `.env.example` to `.env` and change development credentials as needed.
2. Start the stack:

   ```bash
   docker compose up --build
   ```

3. Open:
   - Frontend: http://localhost:3000
   - Backend health: http://localhost:8080/health
   - Worker API health: http://localhost:8000/health
   - Ollama: http://localhost:11434

Ollama models are not downloaded automatically. Pull the configured model after startup when it is first needed.

## Local development

The JavaScript workspace uses pnpm and Turborepo:

```bash
pnpm install
pnpm dev
```

Run the Go and Python services independently from their respective directories. See [docs/architecture.md](docs/architecture.md) for service responsibilities and intended data flow.

## Current scope

Included: build definitions, configuration loading, dependency clients, health endpoints, process entry points, and persistent local volumes.

Deferred: authentication, database models/migrations, upload flows, media processing jobs, transcription pipelines, AI prompts, and product UI.
