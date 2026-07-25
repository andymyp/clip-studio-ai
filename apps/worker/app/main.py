from functools import lru_cache
from pathlib import Path

from fastapi import FastAPI, HTTPException
from fastapi.staticfiles import StaticFiles
from redis import Redis

from .config import get_settings
from .job_store import AnalysisJobStore
from .schemas import AnalysisJob, AnalyzeVideoRequest

settings = get_settings()
media_root = Path(settings.storage_path).resolve() / "output" / "clips"
media_root.mkdir(parents=True, exist_ok=True)

app = FastAPI(
    title="ClipStudio AI Worker API",
    version="0.1.0",
    docs_url="/docs",
    redoc_url=None,
)
app.mount("/media/clips", StaticFiles(directory=media_root), name="clip-media")


@app.get("/health", tags=["system"])
async def health() -> dict[str, str]:
    return {"status": "ok"}


@lru_cache
def get_job_store() -> AnalysisJobStore:
    client = Redis.from_url(settings.redis_url)
    return AnalysisJobStore(client, settings.job_queue_name)


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
