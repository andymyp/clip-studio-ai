from pathlib import Path

from app.config import Settings


def test_storage_path_is_independent_of_process_working_directory() -> None:
    settings = Settings(
        storage_path="../../storage",
        ytdlp_cookie_file="../../storage/secrets/youtube-cookies.txt",
    )

    repository_root = Path(__file__).resolve().parents[3]
    assert Path(settings.storage_path) == repository_root / "storage"
    assert Path(settings.ytdlp_cookie_file) == (
        repository_root / "storage" / "secrets" / "youtube-cookies.txt"
    )
