from fastapi import FastAPI

app = FastAPI(
    title="ClipStudio AI Worker API",
    version="0.1.0",
    docs_url="/docs",
    redoc_url=None,
)


@app.get("/health", tags=["system"])
async def health() -> dict[str, str]:
    return {"status": "ok"}
