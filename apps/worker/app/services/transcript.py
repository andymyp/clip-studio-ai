import html
import json
import re
import subprocess
import sys
import tempfile
from pathlib import Path

from app.schemas import TranscriptSegment


class TranscriptExtractionError(RuntimeError):
    pass


class TranscriptService:
    def __init__(
        self,
        language: str = "en",
        executable: str | None = None,
        js_runtime: str = "node",
        impersonate: str = "chrome",
        cookie_file: str = "../../storage/secrets/youtube-cookies.txt",
        sleep_requests: float = 2,
        sleep_subtitles: float = 5,
    ) -> None:
        self.language = language
        self.executable = executable
        self.js_runtime = js_runtime
        self.impersonate = impersonate
        self.cookie_file = cookie_file
        self.sleep_requests = sleep_requests
        self.sleep_subtitles = sleep_subtitles

    def extract(self, url: str) -> list[TranscriptSegment]:
        with tempfile.TemporaryDirectory(prefix="clipstudio-subs-") as directory:
            output = Path(directory) / "subtitle"
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
                "5",
                "--retry-sleep",
                "http:exp=1:20",
                "--retry-sleep",
                "extractor:exp=1:20",
                "--sleep-requests",
                str(self.sleep_requests),
                "--sleep-subtitles",
                str(self.sleep_subtitles),
                "--skip-download",
                "--write-auto-subs",
                "--write-subs",
                "--sub-langs",
                f"{self.language}.*,en.*,{self.language},en",
                "--sub-format",
                "vtt",
                "--no-playlist",
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
            if completed.returncode != 0:
                if "HTTP Error 429" in completed.stderr:
                    raise TranscriptExtractionError(
                        "YouTube rate-limited subtitle access (HTTP 429). "
                        "Open YouTube in the configured Firefox profile, solve any CAPTCHA, "
                        "then close Firefox and retry."
                    )
                raise TranscriptExtractionError(
                    completed.stderr.strip() or "yt-dlp subtitle extraction failed"
                )
            files = sorted(Path(directory).glob("subtitle*.vtt"))
            if not files:
                raise TranscriptExtractionError("no English subtitles are available")
            return self.parse_vtt(files[0].read_text(encoding="utf-8"))

    def _command_prefix(self) -> list[str]:
        if self.executable:
            return [self.executable]
        return [sys.executable, "-m", "yt_dlp"]

    def _cookie_arguments(self) -> list[str]:
        path = Path(self.cookie_file).expanduser().resolve()
        if not path.is_file():
            raise TranscriptExtractionError(f"yt-dlp cookie file does not exist: {path}")
        return ["--cookies", str(path)]

    @staticmethod
    def parse_vtt(content: str) -> list[TranscriptSegment]:
        segments: list[TranscriptSegment] = []
        blocks = re.split(r"\r?\n\r?\n+", content.replace("\ufeff", "").strip())
        for block in blocks:
            lines = [line.strip() for line in block.splitlines() if line.strip()]
            timing_index = next(
                (index for index, line in enumerate(lines) if "-->" in line),
                None,
            )
            if timing_index is None:
                continue
            timing = lines[timing_index].split("-->")
            if len(timing) != 2:
                continue
            start = _timestamp_seconds(timing[0].strip())
            end = _timestamp_seconds(timing[1].strip().split()[0])
            text = " ".join(lines[timing_index + 1 :])
            text = re.sub(r"<[^>]+>", "", html.unescape(text)).strip()
            text = re.sub(r"\s+", " ", text)
            if not text or end <= start:
                continue
            if segments and segments[-1].text == text and start <= segments[-1].end:
                segments[-1].end = max(segments[-1].end, end)
                continue
            segments.append(TranscriptSegment(text=text, start=start, end=end))
        if not segments:
            raise TranscriptExtractionError("subtitle file contains no usable cues")
        return segments

    @staticmethod
    def to_json(segments: list[TranscriptSegment]) -> str:
        return json.dumps([segment.model_dump() for segment in segments], ensure_ascii=False)


def _timestamp_seconds(value: str) -> float:
    parts = value.replace(",", ".").split(":")
    if len(parts) == 3:
        hours, minutes, seconds = parts
    elif len(parts) == 2:
        hours, minutes, seconds = "0", parts[0], parts[1]
    else:
        raise TranscriptExtractionError(f"invalid VTT timestamp: {value}")
    return int(hours) * 3600 + int(minutes) * 60 + float(seconds)
