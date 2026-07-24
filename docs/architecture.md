# Architecture

## Service boundaries

| Service | Responsibility |
| --- | --- |
| `frontend` | Browser UI and API consumption |
| `backend` | Public API, persistence coordination, SSE streams, and Asynq job production |
| `worker` | Internal Python capability API and media worker foundation |
| `postgres` | Durable application metadata |
| `redis` | Asynq jobs, worker queues, caching, and ephemeral coordination |
| `ollama` | Local language-model inference |

## Intended flow

The frontend calls the Go API. The API persists metadata in PostgreSQL, publishes processing work through Redis, and reports progress over SSE. Python workers consume jobs, use yt-dlp/FFmpeg/Faster Whisper as appropriate, persist artifacts in shared storage, and report state through infrastructure contracts that will be defined with the business layer.

The backend and Python worker deliberately have separate queues at this foundation stage: Asynq establishes the Go-side async boundary, while the Python package supplies a Celery-style Redis consumer. A production job contract and bridge should be selected before business implementation.

## Storage

`storage/input`, `storage/output`, and `storage/temp` are mounted into services at `/app/storage`. Production deployments should replace this local filesystem contract with durable object storage or a managed shared volume.

## Configuration ownership

- `.env.root` controls Compose host ports and infrastructure credentials.
- `.env.fe` contains browser-safe Next.js configuration.
- `.env.be` contains Go API configuration and internal service addresses.
- `.env.worker` is shared by the Python API and background worker processes.

Only the corresponding `.example` templates are committed. PostgreSQL credentials in
`.env.root` and the backend `DATABASE_URL` must remain aligned. Application templates
use `localhost` for terminal-first development; Compose overrides internal addresses
with Docker DNS service names.

## Security notes

The example credentials are development-only. Production deployments should use a secrets manager, terminate TLS at an ingress, isolate internal services, authenticate SSE/API traffic, restrict uploaded media types and sizes, and apply retention policies to generated artifacts.
