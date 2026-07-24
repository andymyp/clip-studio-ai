# ClipStudio AI

ClipStudio AI is a monorepo foundation for an AI-assisted video clipping
platform. It currently provides service boundaries, dependency wiring,
containers, health endpoints, and development tooling. Product business logic
has not been implemented yet.

## Technology

- Frontend: Next.js 15, TypeScript, Tailwind CSS, shadcn/ui, TanStack Query,
  and Zustand
- Backend: Go 1.23, Gin, GORM, PostgreSQL, Redis, Asynq, and SSE
- Worker: Python 3.12, FastAPI, FFmpeg, yt-dlp, and Faster Whisper
- Infrastructure: Docker Compose, PostgreSQL 16, Redis 7, and Ollama
- Monorepo tooling: pnpm and Turborepo

## Repository structure

```text
clip-studio-ai/
|-- frontend/           Next.js application and Dockerfile
|-- backend/            Go API and Dockerfile
|-- worker/             Python API, background worker, and Dockerfile
|-- docker/             Docker documentation
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

- Node.js 20 or later
- pnpm 10
- Go 1.23 or later
- Python 3.12
- Docker Desktop with Docker Compose

Enable the pnpm version declared by the repository:

```powershell
corepack enable
corepack prepare pnpm@10.13.1 --activate
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
- creates `worker/.venv`;
- installs Python worker and development dependencies.

Activate the Python environment in the terminal used to run development
commands:

```powershell
.\worker\.venv\Scripts\Activate.ps1
```

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

Run Python-related commands with `worker/.venv` activated.

## Current scope

Included:

- application shells and process entry points;
- configuration loading;
- infrastructure clients;
- health endpoints;
- SSE foundation;
- container builds and local orchestration;
- persistent development storage.

Deferred:

- authentication and authorization;
- database models and migrations;
- uploads and media ingestion;
- job contracts and processing workflows;
- transcription and AI pipelines;
- prompts and product user interfaces.

See [docs/architecture.md](docs/architecture.md) for the intended service
boundaries and data flow.
