import shutil
import subprocess
import sys
from pathlib import Path


class PartialDownloadError(RuntimeError):
    pass


class PartialDownloaderService:
    def __init__(
        self,
        storage_path: str,
        executable: str | None = None,
        js_runtime: str = "node",
        impersonate: str = "chrome",
        cookie_file: str = "../../storage/secrets/youtube-cookies.txt",
        sleep_requests: float = 2,
    ) -> None:
        self.output_root = Path(storage_path).resolve() / "output" / "clips"
        self.executable = executable
        self.js_runtime = js_runtime
        self.impersonate = impersonate
        self.cookie_file = cookie_file
        self.sleep_requests = sleep_requests

    def download(
        self,
        url: str,
        start_time: float,
        end_time: float,
        job_id: str,
        clip_id: str,
    ) -> Path:
        if start_time < 0 or end_time <= start_time:
            raise ValueError("clip time range is invalid")
        if shutil.which("ffmpeg") is None:
            raise PartialDownloadError(
                "FFmpeg was not found. Install FFmpeg and ensure `ffmpeg` is available on PATH."
            )
        job_directory = self.output_root / job_id
        job_directory.mkdir(parents=True, exist_ok=True)
        output = job_directory / f"{clip_id}.mp4"
        prefix = self._command_prefix()
        command = [
            *prefix,
            "--ignore-config",
            *self._cookie_arguments(),
            "--impersonate",
            self.impersonate,
            "--js-runtimes",
            self.js_runtime,
            "--remote-components",
            "ejs:github",
            "--retries",
            "2",
            "--socket-timeout",
            "20",
            "--retry-sleep",
            "http:exp=1:20",
            "--sleep-requests",
            str(self.sleep_requests),
            "--no-playlist",
            "--download-sections",
            f"*{start_time:.3f}-{end_time:.3f}",
            "--force-keyframes-at-cuts",
            "--merge-output-format",
            "mp4",
            "--format",
            (
                "bv*[height<=1080][vcodec^=avc1]+ba[ext=m4a]/"
                "bv*[height<=1080]+ba/b[height<=1080]/b"
            ),
            "--format-sort",
            "res:1080,fps,hdr:12,vcodec:avc1,acodec:m4a",
            "--output",
            str(output),
            url,
        ]
        completed = subprocess.run(
            command,
            capture_output=True,
            text=True,
            timeout=180,
            check=False,
        )
        if completed.returncode != 0 or not output.exists():
            raise PartialDownloadError(
                completed.stderr.strip() or "partial video download failed"
            )
        return output

    def _command_prefix(self) -> list[str]:
        if self.executable:
            return [self.executable]
        return [sys.executable, "-m", "yt_dlp"]

    def _cookie_arguments(self) -> list[str]:
        path = Path(self.cookie_file).expanduser().resolve()
        if not path.is_file():
            raise PartialDownloadError(f"yt-dlp cookie file does not exist: {path}")
        return ["--cookies", str(path)]
