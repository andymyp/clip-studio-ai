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
        silence_threshold: float = 0.65,
        breath_padding: float = 0.12,
        remove_fillers: bool = True,
    ) -> list[EditInterval]:
        command = [
            "ffmpeg", "-hide_banner", "-i", str(clip), "-af",
            f"silencedetect=noise=-38dB:d={silence_threshold:g}", "-f", "null", "-",
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
            if silence_duration >= silence_threshold:
                silences.append((start + breath_padding, max(start + breath_padding, end - breath_padding)))
        if remove_fillers:
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
        timeline: list[tuple[EditInterval, float]] = []
        elapsed = 0.0
        for interval in intervals:
            timeline.append((interval, elapsed))
            elapsed += interval.end - interval.start

        for segment in transcript:
            text = segment.text.strip()
            if text.lower().strip(".,!? ") in FILLERS:
                continue
            overlaps: list[tuple[float, float, float]] = []
            for interval, offset in timeline:
                start = max(segment.start, interval.start)
                end = min(segment.end, interval.end)
                if end > start:
                    overlaps.append((end - start, offset + start - interval.start, offset + end - interval.start))
            if not overlaps:
                continue
            # A subtitle cue spanning an edit boundary must be emitted once,
            # using the retained side containing most of the spoken cue.
            _duration, mapped_start, mapped_end = max(overlaps, key=lambda item: item[0])
            result.append(
                TranscriptSegment(text=text, start=mapped_start, end=mapped_end)
            )
        return sorted(result, key=lambda item: item.start)


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
