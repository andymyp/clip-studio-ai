from collections.abc import AsyncIterator
from contextlib import asynccontextmanager
from functools import lru_cache
from pathlib import Path

from fastapi import FastAPI, HTTPException
from fastapi.staticfiles import StaticFiles
from redis import Redis

from .config import get_settings
from .job_store import AnalysisJobStore
from .render_store import RenderJobStore
from .schemas import AnalysisJob, AnalyzeVideoRequest, RenderJobState, RenderRequest

settings = get_settings()
redis_client = Redis.from_url(settings.redis_url)
media_root = Path(settings.storage_path).resolve() / "output" / "clips"
media_root.mkdir(parents=True, exist_ok=True)
rendered_root = Path(settings.storage_path).resolve() / "output" / "rendered"
rendered_root.mkdir(parents=True, exist_ok=True)

@asynccontextmanager
async def lifespan(_app: FastAPI) -> AsyncIterator[None]:
    yield
    redis_client.close()


app = FastAPI(
    title="ClipStudio AI Worker API",
    version="0.1.0",
    docs_url="/docs",
    redoc_url=None,
    lifespan=lifespan,
)
app.mount("/media/clips", StaticFiles(directory=media_root), name="clip-media")
app.mount("/media/rendered", StaticFiles(directory=rendered_root), name="rendered-media")


@app.get("/health", tags=["system"])
async def health() -> dict[str, str]:
    return {"status": "ok"}


@lru_cache
def get_job_store() -> AnalysisJobStore:
    return AnalysisJobStore(redis_client, settings.job_queue_name)


@lru_cache
def get_render_store() -> RenderJobStore:
    return RenderJobStore(redis_client, settings.render_queue_name)


@app.post("/jobs", response_model=AnalysisJob, status_code=202, tags=["analysis"])
async def create_job(payload: AnalyzeVideoRequest) -> AnalysisJob:
    if payload.platform != "youtube":
        raise HTTPException(status_code=400, detail="subtitle extraction currently supports YouTube")
    return get_job_store().create(payload)


@app.get("/jobs/{job_id}", response_model=AnalysisJob, tags=["analysis"])
async def get_job(job_id: str) -> AnalysisJob:
    job = get_job_store().get(job_id)
    if job is None:
        raise HTTPException(status_code=404, detail="analysis job not found")
    return job


@app.get(
    "/reviews/{user_id}/{external_id}",
    response_model=AnalysisJob,
    tags=["analysis"],
)
async def get_review(user_id: str, external_id: str) -> AnalysisJob:
    job = get_job_store().get_review(user_id, "youtube", external_id)
    if job is None:
        raise HTTPException(status_code=404, detail="analysis review not found")
    return job


@app.post("/render-jobs", response_model=RenderJobState, status_code=202, tags=["render"])
async def create_render_job(payload: RenderRequest) -> RenderJobState:
    return get_render_store().create(payload)


@app.get("/render-jobs/{job_id}", response_model=RenderJobState, tags=["render"])
async def get_render_job(job_id: str) -> RenderJobState:
    job = get_render_store().get(job_id)
    if job is None:
        raise HTTPException(status_code=404, detail="render job not found")
    return job


@app.post("/render-jobs/{job_id}/retry", response_model=RenderJobState, tags=["render"])
async def retry_render_job(job_id: str) -> RenderJobState:
    job = get_render_store().retry(job_id)
    if job is None:
        raise HTTPException(status_code=404, detail="render job not found")
    return job
