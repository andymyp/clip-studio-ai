from pathlib import Path
from unittest.mock import patch

from app.schemas import EditInterval, TranscriptSegment
from app.services.retention import RetentionEditService


def test_removes_dead_air_and_filler_only_segments(tmp_path: Path) -> None:
    stderr = (
        "[silencedetect] silence_end: 3.000 | silence_duration: 1.000\n"
    )
    completed = type("Result", (), {"stderr": stderr})()
    transcript = [
        TranscriptSegment(text="A strong hook", start=0, end=1.5),
        TranscriptSegment(text="um", start=4, end=4.5),
        TranscriptSegment(text="The payoff", start=5, end=8),
    ]
    with patch("app.services.retention.subprocess.run", return_value=completed):
        intervals = RetentionEditService().plan(tmp_path / "clip.mp4", 8, transcript)

    assert all(not (item.start < 2.5 and item.end > 2.5) for item in intervals)
    assert all(not (item.start < 4.25 and item.end > 4.25) for item in intervals)


def test_preserves_intro_and_outro_context_buffers(tmp_path: Path) -> None:
    completed = type(
        "Result",
        (),
        {"stderr": "silence_end: 1.500 | silence_duration: 1.500\nsilence_end: 10.000 | silence_duration: 1.000"},
    )()
    with patch("app.services.retention.subprocess.run", return_value=completed):
        intervals = RetentionEditService().plan(
            tmp_path / "clip.mp4",
            10,
            preserve_start=2,
            preserve_end=1.2,
        )
    assert intervals[0].start == 0
    assert intervals[0].end >= 2
    assert intervals[-1].end == 10


def test_remap_emits_cue_spanning_edit_boundary_only_once() -> None:
    result = RetentionEditService().remap(
        [
            TranscriptSegment(
                text="Never repeat this subtitle",
                start=1.8,
                end=2.4,
            )
        ],
        [EditInterval(start=0, end=2), EditInterval(start=2.2, end=4)],
    )

    assert len(result) == 1
    assert result[0].text == "Never repeat this subtitle"
