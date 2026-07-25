import re
import subprocess
from pathlib import Path

from app.schemas import EditInterval, TranscriptSegment

SILENCE_END = re.compile(r"silence_end: ([\d.]+)")
SILENCE_DURATION = re.compile(r"silence_duration: ([\d.]+)")
FILLERS = {"um", "uh", "erm", "hmm", "like", "you know", "i mean"}


class RetentionEditService:
    """Build a conservative cut list that removes only retention-killing dead air."""

    def plan(
        self,
        clip: Path,
        duration: float,
        transcript: list[TranscriptSegment] | None = None,
        preserve_start: float = 0,
        preserve_end: float = 0,
    ) -> list[EditInterval]:
        command = [
            "ffmpeg", "-hide_banner", "-i", str(clip), "-af",
            "silencedetect=noise=-38dB:d=0.65", "-f", "null", "-",
        ]
        completed = subprocess.run(
            command, capture_output=True, text=True, timeout=120, check=False
        )
        silences: list[tuple[float, float]] = []
        lines = completed.stderr.splitlines()
        for index, line in enumerate(lines):
            end_match = SILENCE_END.search(line)
            duration_match = SILENCE_DURATION.search(line)
            if not end_match or not duration_match:
                continue
            end = float(end_match.group(1))
            silence_duration = float(duration_match.group(1))
            start = max(0, end - silence_duration)
            # Retain a natural 120 ms breath on both sides.
            if silence_duration >= 0.65:
                silences.append((start + 0.12, end - 0.12))
        for segment in transcript or []:
            if segment.text.lower().strip(".,!? ") in FILLERS:
                silences.append((segment.start, segment.end))
        protected_end = max(0, duration - preserve_end)
        removable = [
            (max(start, preserve_start), min(end, protected_end))
            for start, end in silences
            if min(end, protected_end) > max(start, preserve_start)
        ]
        return _invert(removable, duration)

    def remap(
        self,
        transcript: list[TranscriptSegment],
        intervals: list[EditInterval],
    ) -> list[TranscriptSegment]:
        result: list[TranscriptSegment] = []
        elapsed = 0.0
        for interval in intervals:
            for segment in transcript:
                start = max(segment.start, interval.start)
                end = min(segment.end, interval.end)
                if end <= start:
                    continue
                text = segment.text.strip()
                if text.lower().strip(".,!? ") in FILLERS:
                    continue
                result.append(
                    TranscriptSegment(
                        text=text,
                        start=elapsed + start - interval.start,
                        end=elapsed + end - interval.start,
                    )
                )
            elapsed += interval.end - interval.start
        return result


def _invert(cuts: list[tuple[float, float]], duration: float) -> list[EditInterval]:
    intervals: list[EditInterval] = []
    cursor = 0.0
    for start, end in sorted(cuts):
        start, end = max(cursor, start), min(duration, end)
        if start - cursor >= 0.25:
            intervals.append(EditInterval(start=cursor, end=start))
        cursor = max(cursor, end)
    if duration - cursor >= 0.25:
        intervals.append(EditInterval(start=cursor, end=duration))
    return intervals or [EditInterval(start=0, end=duration)]
