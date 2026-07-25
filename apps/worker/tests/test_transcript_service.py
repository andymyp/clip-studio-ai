import sys
from pathlib import Path

from app.services.transcript import TranscriptService


def test_parse_vtt_to_segments() -> None:
    content = """WEBVTT

00:00:00.000 --> 00:00:02.500
<c>Hello world</c>

00:00:02.500 --> 00:00:05.000
This is a useful insight.
"""

    segments = TranscriptService.parse_vtt(content)

    assert [segment.model_dump() for segment in segments] == [
        {"text": "Hello world", "start": 0.0, "end": 2.5},
        {"text": "This is a useful insight.", "start": 2.5, "end": 5.0},
    ]


def test_uses_yt_dlp_from_active_python_environment() -> None:
    assert TranscriptService()._command_prefix() == [sys.executable, "-m", "yt_dlp"]


def test_uses_configured_cookie_file(tmp_path: Path) -> None:
    cookie_file = tmp_path / "cookies.txt"
    cookie_file.write_text("# Netscape HTTP Cookie File\n", encoding="utf-8")
    service = TranscriptService(cookie_file=str(cookie_file))

    assert service._cookie_arguments() == ["--cookies", str(cookie_file.resolve())]
