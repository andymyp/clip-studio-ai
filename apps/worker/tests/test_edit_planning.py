from app.schemas import EditInterval, TranscriptSegment
from app.worker import _semantic_clip_bounds, _visual_beats


def test_semantic_bounds_include_complete_sentence_and_context() -> None:
    transcript = [
        TranscriptSegment(text="First setup.", start=10, end=13),
        TranscriptSegment(text="The core thought continues", start=13, end=17),
        TranscriptSegment(text="until this payoff.", start=17, end=21),
    ]

    start, end = _semantic_clip_bounds(14, 18, transcript, 2, 1.2)

    assert start == 11
    assert end == 22.2


def test_visual_beats_prefer_sentence_boundaries() -> None:
    transcript = [
        TranscriptSegment(text="One complete thought.", start=0, end=6),
        TranscriptSegment(text="Another complete thought.", start=6, end=12),
    ]

    beats = _visual_beats([EditInterval(start=0, end=12)], transcript)

    assert beats == [
        EditInterval(start=0, end=6),
        EditInterval(start=6, end=12),
    ]
