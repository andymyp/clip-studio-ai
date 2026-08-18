from functools import lru_cache
from pathlib import Path

from pydantic import field_validator
from pydantic_settings import BaseSettings, SettingsConfigDict

WORKER_ROOT = Path(__file__).resolve().parents[1]


class Settings(BaseSettings):
    app_env: str = "development"
    redis_url: str = "redis://localhost:6379/0"
    ollama_host: str = "http://localhost:11434"
    ollama_model: str = "qwen2.5:7b"
    storage_path: str = "../../storage"
    whisper_model: str = "small"
    transcript_language: str = "en"
    ytdlp_js_runtime: str = "node"
    ytdlp_impersonate: str = "chrome"
    ytdlp_cookie_file: str = "../../storage/secrets/youtube-cookies.txt"
    ytdlp_sleep_requests: float = 2
    ytdlp_sleep_subtitles: float = 5
    clip_min_duration: float = 20
    clip_max_duration: float = 60
    clip_max_candidates: int = 5
    clip_context_before: float = 2.0
    clip_context_after: float = 1.2
    job_queue_name: str = "clipstudio:analysis:queue"
    render_queue_name: str = "clipstudio:render:queue"
    review_cleanup_interval_seconds: int = 300

    # Support running from the repository root or apps/worker directory.
    model_config = SettingsConfigDict(
        env_file=(".env.worker", "../.env.worker", "../../.env.worker"),
        extra="ignore",
    )

    @field_validator("storage_path", "ytdlp_cookie_file")
    @classmethod
    def resolve_worker_path(cls, value: str) -> str:
        if not value:
            return value
        path = Path(value).expanduser()
        if not path.is_absolute():
            path = WORKER_ROOT / path
        return str(path.resolve())


@lru_cache
def get_settings() -> Settings:
    return Settings()
