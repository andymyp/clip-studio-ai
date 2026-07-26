import re
import unicodedata
from pathlib import Path
from typing import ClassVar

from app.schemas import TranscriptSegment

ALLOWED_PUNCTUATION = set(".,!?;:'\"-()[]") | {
    "\u2013", "\u2014", "\u2018", "\u2019", "\u201c", "\u201d", "\u2026"
}

class SubtitleGenerator:
    IMPORTANT: ClassVar[set[str]] = {
        "why", "how", "never", "best", "secret", "important", "problem", "stop",
        "truth", "mistake", "change", "remember",
    }

    def generate(
        self,
        segments: list[TranscriptSegment],
        output: Path,
        important_phrases: list[str] | None = None,
        platform_profile: str = "youtube",
    ) -> Path:
        output.parent.mkdir(parents=True, exist_ok=True)
        phrase_words = {
            word.lower()
            for phrase in (important_phrases or [])
            for word in re.findall(r"\w+", phrase)
            if len(word) >= 5
        }
        events: list[str] = []
        for segment in _deduplicate_segments(segments):
            words = _sanitize_caption(segment.text).split()
            if not words:
                continue
            chunks = _caption_chunks(words)
            chunk_durations = _chunk_durations(
                chunks, max(0.01, segment.end - segment.start)
            )
            elapsed = 0.0
            for chunk, duration in zip(chunks, chunk_durations, strict=True):
                start = segment.start + elapsed
                end = min(segment.end, start + duration)
                elapsed += duration
                events.append(
                    f"Dialogue: 0,{_ass_time(start)},{_ass_time(end)},"
                    f"Shorts,,0,0,0,,{_style_chunk(chunk, phrase_words)}"
                )
        output.write_text(
            _header(platform_profile) + "\n".join(events) + "\n",
            encoding="utf-8-sig",
        )
        return output


def _caption_chunks(words: list[str], maximum_words: int = 4, maximum_chars: int = 28) -> list[list[str]]:
    chunks: list[list[str]] = []
    current: list[str] = []
    for word in words:
        candidate = " ".join([*current, word])
        clause_end = bool(current and re.search(r"[,;:!?]$", current[-1]))
        if current and (
            len(current) >= maximum_words
            or len(candidate) > maximum_chars
            or clause_end
        ):
            chunks.append(current)
            current = []
        current.append(word)
    if current:
        chunks.append(current)
    if len(chunks) > 1 and len(chunks[-1]) == 1:
        merged = [*chunks[-2], *chunks[-1]]
        if len(merged) <= maximum_words + 1 and len(" ".join(merged)) <= maximum_chars:
            chunks[-2:] = [merged]
    return chunks


def _deduplicate_segments(
    segments: list[TranscriptSegment],
) -> list[TranscriptSegment]:
    result: list[TranscriptSegment] = []
    for segment in sorted(segments, key=lambda item: (item.start, item.end)):
        text = _sanitize_caption(segment.text)
        if not text:
            continue
        words = text.split()
        if result:
            previous = result[-1]
            previous_text = _sanitize_caption(previous.text)
            if (
                text.casefold() == previous_text.casefold()
                and segment.start <= previous.end + 1
            ):
                previous.end = max(previous.end, segment.end)
                continue
            previous_words = previous_text.split()
            overlap = _word_overlap(previous_words, words)
            if overlap and segment.start <= previous.end + 1:
                words = words[overlap:]
                if not words:
                    previous.end = max(previous.end, segment.end)
                    continue
                text = " ".join(words)
        result.append(segment.model_copy(update={"text": text}))
    return result


def _word_overlap(previous: list[str], current: list[str]) -> int:
    maximum = min(len(previous), len(current))
    for size in range(maximum, 0, -1):
        left = [word.casefold().strip(".,!?;:") for word in previous[-size:]]
        right = [word.casefold().strip(".,!?;:") for word in current[:size]]
        if left == right:
            return size
    return 0


def _chunk_durations(chunks: list[list[str]], total: float) -> list[float]:
    desired = [max(0.7, len(chunk) / 3.0) for chunk in chunks]
    desired_total = sum(desired)
    if desired_total <= total:
        extra = total - desired_total
        weight = sum(len(chunk) for chunk in chunks)
        return [
            duration + extra * len(chunk) / max(1, weight)
            for chunk, duration in zip(chunks, desired, strict=True)
        ]
    scale = total / max(0.01, desired_total)
    return [duration * scale for duration in desired]


def _sanitize_caption(text: str) -> str:
    value: list[str] = []
    for character in text:
        category = unicodedata.category(character)
        if category[0] in {"L", "N"} or character in ALLOWED_PUNCTUATION or character.isspace():
            value.append(character)
    return " ".join("".join(value).split())


def _style_chunk(words: list[str], phrase_words: set[str]) -> str:
    emphasis_index = next(
        (
            index
            for index, word in enumerate(words)
            if re.sub(r"[^\w]", "", word).lower() in SubtitleGenerator.IMPORTANT
            or re.sub(r"[^\w]", "", word).lower() in phrase_words
        ),
        -1,
    )
    styled: list[str] = []
    for index, word in enumerate(words):
        escaped = word.replace("\\", r"\\").replace("{", r"\{").replace("}", r"\}")
        if index == emphasis_index:
            styled.append(r"{\c&H0000D7FF&\b1}" + escaped + r"{\c&H00FFFFFF&\b1}")
        else:
            styled.append(escaped)
    return " ".join(styled)


def _ass_time(seconds: float) -> str:
    hours = int(seconds // 3600)
    minutes = int(seconds % 3600 // 60)
    remainder = seconds % 60
    return f"{hours}:{minutes:02d}:{remainder:05.2f}"


def _header(platform_profile: str = "youtube") -> str:
    margin = {"youtube": 310, "tiktok": 400, "instagram": 360}.get(
        platform_profile, 310
    )
    return f"""[Script Info]
ScriptType: v4.00+
PlayResX: 1080
PlayResY: 1920
WrapStyle: 2
ScaledBorderAndShadow: yes

[V4+ Styles]
Format: Name,Fontname,Fontsize,PrimaryColour,SecondaryColour,OutlineColour,BackColour,Bold,Italic,Underline,StrikeOut,ScaleX,ScaleY,Spacing,Angle,BorderStyle,Outline,Shadow,Alignment,MarginL,MarginR,MarginV,Encoding
Style: Shorts,Arial,64,&H00FFFFFF,&H00FFFFFF,&H00101010,&H70000000,-1,0,0,0,100,100,0,0,1,7,2,2,90,90,{margin},1

[Events]
Format: Layer,Start,End,Style,Name,MarginL,MarginR,MarginV,Effect,Text
"""
