from unittest.mock import patch

from app.schemas import RankedClip, TranscriptSegment
from app.services.clip_ranking import ClipRankingService


def test_analyze_long_transcript_in_windows() -> None:
    service = ClipRankingService(
        "http://ollama",
        maximum_candidates=2,
        window_duration=60,
        window_overlap=10,
        maximum_analysis_windows=4,
    )
    transcript = [
        TranscriptSegment(text=f"Segment {index}", start=index * 10, end=index * 10 + 10)
        for index in range(18)
    ]
    responses = [
        [RankedClip(start=10, end=38, score=92, reason="Strong hook")],
        [RankedClip(start=65, end=90, score=88, reason="Useful insight")],
        [RankedClip(start=120, end=145, score=75, reason="Curiosity")],
        [],
    ]

    with patch.object(service, "_analyze_window", side_effect=responses) as analyze_window:
        clips = service.analyze(transcript)

    assert analyze_window.call_count == 4
    assert [clip.score for clip in clips] == [92, 88]


def test_preselects_only_best_analysis_windows() -> None:
    service = ClipRankingService("http://ollama", maximum_analysis_windows=1)
    ordinary = [TranscriptSegment(text="ordinary words", start=0, end=30)]
    strong = [
        TranscriptSegment(
            text="Why this surprising mistake is important?",
            start=60,
            end=90,
        )
    ]

    assert service._select_windows([ordinary, strong]) == [strong]


def test_normalizes_slightly_short_candidate() -> None:
    service = ClipRankingService("http://ollama", minimum_duration=20)
    candidate = RankedClip(start=30, end=48, score=90, reason="Dense explanation")

    normalized = service._normalize(candidate, transcript_end=100)

    assert normalized is not None
    assert normalized.start == 30
    assert normalized.end == 50


def test_heuristic_fallback_returns_non_overlapping_candidates() -> None:
    service = ClipRankingService(
        "http://ollama",
        maximum_candidates=2,
        minimum_duration=20,
        maximum_duration=60,
    )
    transcript = [
        TranscriptSegment(
            text=f"Why this important idea solves the problem number {index}",
            start=index * 5,
            end=index * 5 + 5,
        )
        for index in range(30)
    ]

    with patch.object(service, "_analyze_window", return_value=[]):
        clips = service.analyze(transcript)

    assert len(clips) == 2
    assert all(20 <= clip.end - clip.start <= 60 for clip in clips)
    assert clips[0].end <= clips[1].start or clips[1].end <= clips[0].start
