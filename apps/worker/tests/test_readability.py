from app.schemas import TranscriptSegment
from app.worker import _readability_speed


def test_keeps_natural_dialogue_at_original_speed() -> None:
    transcript = [TranscriptSegment(text="one two three four five six", start=0, end=3)]
    assert _readability_speed(transcript, 0, 3) == 1


def test_gently_slows_fast_dialogue_with_a_safe_floor() -> None:
    fast = [TranscriptSegment(text=" ".join(["word"] * 40), start=0, end=8)]
    extreme = [TranscriptSegment(text=" ".join(["word"] * 100), start=0, end=8)]
    assert _readability_speed(fast, 0, 8) == 0.85
    assert _readability_speed(extreme, 0, 8) == 0.85
