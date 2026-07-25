from pathlib import Path

from app.schemas import TranscriptSegment
from app.services.subtitle import SubtitleGenerator


def test_generates_shorts_ass_with_word_timing_and_emphasis(tmp_path: Path) -> None:
    output = SubtitleGenerator().generate(
        [TranscriptSegment(text="Why this works", start=0, end=3)],
        tmp_path / "subtitle.ass",
    )

    content = output.read_text(encoding="utf-8-sig")
    assert "PlayResX: 1080" in content
    assert r"{\k" not in content
    assert r"{\c&H0000D7FF&\b1}Why" in content
    assert "Dialogue: 0,0:00:00.00,0:00:03.00,Shorts" in content


def test_long_caption_is_split_into_safe_short_phrases(tmp_path: Path) -> None:
    output = SubtitleGenerator().generate(
        [
            TranscriptSegment(
                text="This sentence is intentionally much too long to fit inside one vertical caption",
                start=0,
                end=6,
            )
        ],
        tmp_path / "subtitle.ass",
    )
    dialogues = [
        line
        for line in output.read_text(encoding="utf-8-sig").splitlines()
        if line.startswith("Dialogue:")
    ]
    assert len(dialogues) >= 3
    assert all(len(line.rsplit(",,", 1)[-1].split()) <= 4 for line in dialogues)


def test_removes_non_punctuation_symbols_from_captions(tmp_path: Path) -> None:
    output = SubtitleGenerator().generate(
        [TranscriptSegment(text="Great idea! 🚀 Save $50 & grow → now.", start=0, end=4)],
        tmp_path / "subtitle.ass",
    )
    content = output.read_text(encoding="utf-8-sig")
    dialogue_text = "\n".join(
        line.rsplit(",,", 1)[-1]
        for line in content.splitlines()
        if line.startswith("Dialogue:")
    )
    assert "Great idea!" in content
    assert "now." in content
    assert "🚀" not in content
    assert "$" not in dialogue_text
    assert "&" not in dialogue_text
    assert "→" not in dialogue_text
