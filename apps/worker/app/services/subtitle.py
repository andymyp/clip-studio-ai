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
    ) -> Path:
        output.parent.mkdir(parents=True, exist_ok=True)
        phrase_words = {
            word.lower()
            for phrase in (important_phrases or [])
            for word in re.findall(r"\w+", phrase)
            if len(word) >= 5
        }
        events: list[str] = []
        for segment in segments:
            words = _sanitize_caption(segment.text).split()
            if not words:
                continue
            chunks = _caption_chunks(words)
            word_duration = (segment.end - segment.start) / len(words)
            consumed = 0
            for chunk in chunks:
                start = segment.start + consumed * word_duration
                end = min(segment.end, start + len(chunk) * word_duration)
                consumed += len(chunk)
                events.append(
                    f"Dialogue: 0,{_ass_time(start)},{_ass_time(end)},"
                    f"Shorts,,0,0,0,,{_style_chunk(chunk, phrase_words)}"
                )
        output.write_text(_header() + "\n".join(events) + "\n", encoding="utf-8-sig")
        return output


def _caption_chunks(words: list[str], maximum_words: int = 4, maximum_chars: int = 28) -> list[list[str]]:
    chunks: list[list[str]] = []
    current: list[str] = []
    for word in words:
        candidate = " ".join([*current, word])
        if current and (len(current) >= maximum_words or len(candidate) > maximum_chars):
            chunks.append(current)
            current = []
        current.append(word)
    if current:
        chunks.append(current)
    return chunks


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


def _header() -> str:
    return """[Script Info]
ScriptType: v4.00+
PlayResX: 1080
PlayResY: 1920
WrapStyle: 2
ScaledBorderAndShadow: yes

[V4+ Styles]
Format: Name,Fontname,Fontsize,PrimaryColour,SecondaryColour,OutlineColour,BackColour,Bold,Italic,Underline,StrikeOut,ScaleX,ScaleY,Spacing,Angle,BorderStyle,Outline,Shadow,Alignment,MarginL,MarginR,MarginV,Encoding
Style: Shorts,Arial,64,&H00FFFFFF,&H00FFFFFF,&H00101010,&H70000000,-1,0,0,0,100,100,0,0,1,7,2,2,90,90,310,1

[Events]
Format: Layer,Start,End,Style,Name,MarginL,MarginR,MarginV,Effect,Text
"""
