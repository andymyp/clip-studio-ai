import json
import re
import subprocess
from pathlib import Path

from app.schemas import QualityReport


class RenderQualityError(RuntimeError):
    pass


class RenderQualityService:
    def inspect(self, output: Path, expected_duration: float) -> QualityReport:
        command = [
            "ffprobe", "-v", "error", "-show_streams", "-show_format",
            "-of", "json", str(output),
        ]
        completed = subprocess.run(
            command, capture_output=True, text=True, timeout=60, check=False
        )
        if completed.returncode != 0:
            raise RenderQualityError(completed.stderr.strip() or "ffprobe failed")
        try:
            payload = json.loads(completed.stdout)
            streams = payload["streams"]
            video = next(item for item in streams if item["codec_type"] == "video")
            audio = next((item for item in streams if item["codec_type"] == "audio"), None)
            duration = float(payload["format"]["duration"])
            video_duration = float(video.get("duration", duration))
            audio_duration = float(audio.get("duration", duration)) if audio else 0
        except (KeyError, TypeError, ValueError, StopIteration, json.JSONDecodeError) as exc:
            raise RenderQualityError("invalid ffprobe output") from exc

        drift = abs(video_duration - audio_duration) if audio else duration
        warnings: list[str] = []
        scan = subprocess.run(
            [
                "ffmpeg", "-hide_banner", "-i", str(output),
                "-vf", "blackdetect=d=0.25:pix_th=0.10,freezedetect=n=-45dB:d=1.5",
                "-an", "-f", "null", "-",
            ],
            capture_output=True,
            text=True,
            timeout=180,
            check=False,
        )
        black_seconds = sum(
            float(value)
            for value in re.findall(r"black_duration:([\d.]+)", scan.stderr)
        )
        freeze_starts = [
            float(value) for value in re.findall(r"freeze_start: ([\d.]+)", scan.stderr)
        ]
        freeze_ends = [
            float(value) for value in re.findall(r"freeze_end: ([\d.]+)", scan.stderr)
        ]
        frozen_seconds = sum(
            max(0, end - start)
            for start, end in zip(freeze_starts, freeze_ends, strict=False)
        )
        black_ratio = black_seconds / max(duration, 0.01)
        frozen_ratio = frozen_seconds / max(duration, 0.01)
        if abs(duration - expected_duration) > 0.35:
            warnings.append("output duration differs from render plan")
        if drift > 0.12:
            warnings.append("audio/video drift exceeds 120 ms")
        if black_ratio > 0.05:
            warnings.append("black frames exceed 5% of output")
        if frozen_ratio > 0.15:
            warnings.append("frozen frames exceed 15% of output")
        report = QualityReport(
            width=int(video.get("width", 0)),
            height=int(video.get("height", 0)),
            duration=round(duration, 3),
            has_audio=audio is not None,
            audio_video_drift=round(drift, 3),
            black_frame_ratio=round(black_ratio, 4),
            frozen_frame_ratio=round(frozen_ratio, 4),
            warnings=warnings,
        )
        report.passed = (
            report.width == 1080
            and report.height == 1920
            and report.has_audio
            and drift <= 0.12
            and abs(duration - expected_duration) <= 0.35
            and black_ratio <= 0.05
            and frozen_ratio <= 0.15
        )
        if not report.passed:
            raise RenderQualityError("; ".join(warnings) or "render media validation failed")
        return report
