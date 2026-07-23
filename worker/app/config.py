from functools import lru_cache

from pydantic_settings import BaseSettings, SettingsConfigDict


class Settings(BaseSettings):
    app_env: str = "development"
    redis_url: str = "redis://localhost:6379/0"
    ollama_host: str = "http://localhost:11434"
    storage_path: str = "./storage"
    whisper_model: str = "small"

    model_config = SettingsConfigDict(env_file=".env", extra="ignore")


@lru_cache
def get_settings() -> Settings:
    return Settings()
