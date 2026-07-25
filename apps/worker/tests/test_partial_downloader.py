from pathlib import Path
from subprocess import CompletedProcess
from unittest.mock import patch

from app.services.partial_downloader import PartialDownloaderService


def test_download_uses_section_only(tmp_path: Path) -> None:
    cookie_file = tmp_path / "cookies.txt"
    cookie_file.write_text("# Netscape HTTP Cookie File\n", encoding="utf-8")
    service = PartialDownloaderService(str(tmp_path), cookie_file=str(cookie_file))

    def fake_run(command: list[str], **_kwargs: object) -> CompletedProcess[str]:
        output = Path(command[command.index("--output") + 1])
        output.touch()
        return CompletedProcess(command, 0, "", "")

    with (
        patch("app.services.partial_downloader.shutil.which", return_value="ffmpeg"),
        patch("app.services.partial_downloader.subprocess.run", side_effect=fake_run) as run,
    ):
        output = service.download("https://youtube.com/watch?v=test", 80, 120, "job", "clip")

    command = run.call_args.args[0]
    assert "--download-sections" in command
    assert command[command.index("--download-sections") + 1] == "*80.000-120.000"
    assert "--download-sections" in command
    assert "--socket-timeout" in command
    assert command[command.index("--cookies") + 1] == str(cookie_file.resolve())
    assert command[command.index("--impersonate") + 1] == "chrome"
    assert output.name == "clip.mp4"
