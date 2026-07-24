# Docker

Container build definitions live beside each service:

- `frontend/Dockerfile`
- `backend/Dockerfile`
- `worker/Dockerfile`

The repository-level `docker-compose.yml` owns local orchestration, networking,
health checks, dependency ordering, and persistent volumes.
