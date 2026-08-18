from pathlib import Path

from app.schemas import TranscriptSegment
from app.services.subtitle import SubtitleGenerator


def test_removes_exact_and_progressive_duplicate_caption_cues(tmp_path: Path) -> None:
    output = SubtitleGenerator().generate(
        [
            TranscriptSegment(text="This idea changes", start=0, end=1.5),
            TranscriptSegment(text="This idea changes", start=1.2, end=2),
            TranscriptSegment(text="changes how you work", start=1.8, end=3.5),
        ],
        tmp_path / "subtitle.ass",
    )
    dialogue = " ".join(
        line.rsplit(",,", 1)[-1]
        for line in output.read_text(encoding="utf-8-sig").splitlines()
        if line.startswith("Dialogue:")
    )

    assert dialogue.count("This idea changes") == 1
    assert "changes changes" not in dialogue
    assert "you work" in dialogue
