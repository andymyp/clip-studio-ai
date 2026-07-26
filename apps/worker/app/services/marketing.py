import json
import re
from urllib import error, request

from pydantic import BaseModel, Field

from app.schemas import MarketingMetadata, TranscriptSegment


class PackagingCandidates(BaseModel):
    main_topic: str
    main_claim: str
    payoff: str
    titles: list[str] = Field(min_length=3, max_length=3)
    hooks: list[str] = Field(min_length=3, max_length=3)
    description: str
    hashtags: list[str] = Field(default_factory=list)


class MarketingGenerator:
    def __init__(self, ollama_host: str, model: str) -> None:
        self.endpoint = ollama_host.rstrip("/") + "/api/chat"
        self.model = model

    def generate(
        self,
        transcript: list[TranscriptSegment],
        source_title: str = "",
    ) -> MarketingMetadata:
        text = " ".join(segment.text for segment in transcript)
        schema = PackagingCandidates.model_json_schema()
        payload = json.dumps(
            {
                "model": self.model,
                "stream": False,
                "format": schema,
                "messages": [
                    {
                        "role": "system",
                        "content": (
                            "Act as a senior short-form video editor. Extract the clip's real "
                            "MAIN_TOPIC: the specific subject in 1-4 words. "
                            "MAIN_CLAIM: the speaker's central complete claim. "
                            "PAYOFF: the exact insight delivered by the end. Base all packaging "
                            "on this brief and the full clip, not only its first sentence. "
                            "TITLES: provide exactly 3 accurate, specific publishing titles. "
                            "Each must be 5-10 words, front-load the important idea, communicate "
                            "the payoff, and sound natural outside the transcript. "
                            "HOOKS: provide exactly 3 separate 3-6 word opening-card headlines. "
                            "Each must create an open loop that the clip immediately resolves. "
                            "A hook is not the publishing title and must not copy any transcript "
                            "sentence or reuse a title phrase. "
                            "DESCRIPTION: 1-2 complete sentences explaining the unique value "
                            "without copying a title, hook, or transcript sentence. "
                            "HASHTAGS: 3-5 relevant lowercase tags without # symbols. "
                            "Never use Markdown, labels, emoji, ALL CAPS, clickbait, generic "
                            "phrases, sentence fragments, repeated words, or invented claims. "
                            "Return JSON only."
                        ),
                    },
                    {
                        "role": "user",
                        "content": (
                            f"Source video title: {source_title or 'Unknown'}\n"
                            f"Selected clip transcript:\n{text[:6000]}"
                        ),
                    },
                ],
                "options": {"temperature": 0.55, "num_ctx": 3072, "num_predict": 512},
            }
        ).encode()
        http_request = request.Request(
            self.endpoint,
            data=payload,
            headers={"Content-Type": "application/json"},
            method="POST",
        )
        try:
            with request.urlopen(http_request, timeout=90) as response:
                content = json.load(response)["message"]["content"]
            candidates = PackagingCandidates.model_validate_json(content)
            return _select(candidates, text, source_title)
        except (error.URLError, TimeoutError, KeyError, ValueError, json.JSONDecodeError):
            return _fallback(text, source_title)


def _select(
    candidates: PackagingCandidates,
    transcript: str,
    source_title: str = "",
) -> MarketingMetadata:
    context = (
        f"{candidates.main_topic} {candidates.main_claim} "
        f"{candidates.payoff} {source_title}"
    )
    titles = [
        value
        for raw in candidates.titles
        if (value := _clean(raw, 80)) and _valid_title(value, transcript, context)
    ]
    titles.sort(key=lambda value: _context_score(value, context), reverse=True)
    title = titles[0] if titles else _fallback_title(transcript, candidates.main_topic)
    hooks = [
        value
        for raw in candidates.hooks
        if (value := _clean(raw, 48))
        and _valid_hook(value, title, transcript, context)
    ]
    hooks.sort(key=lambda value: _context_score(value, context), reverse=True)
    hook = hooks[0] if hooks else _fallback_hook(
        title, transcript, candidates.main_topic
    )
    return _normalize(
        MarketingMetadata(
            title=title,
            description=candidates.description,
            hashtags=candidates.hashtags,
            hook=hook,
        ),
        transcript,
        context,
    )


def _normalize(
    metadata: MarketingMetadata,
    transcript: str,
    context: str = "",
) -> MarketingMetadata:
    title = _clean(metadata.title, 80) or _fallback_title(transcript, context)
    hook = _clean(metadata.hook, 48)
    description = _clean(metadata.description, 300)
    topic = _topic(transcript)

    if not _valid_title(title, transcript, context):
        title = _fallback_title(transcript, context)
    if not _valid_hook(hook, title, transcript, context):
        hook = _fallback_hook(title, transcript, context)
    if _similar(description, title) or _similar(description, hook) or len(description) < 35:
        description = (
            f"This clip explores {topic.lower()} and the practical idea behind it. "
            "Watch the full moment for the key takeaway."
        )

    hashtags = [
        re.sub(r"[^a-z0-9]", "", value.lower().lstrip("#"))
        for value in metadata.hashtags
    ]
    hashtags = list(dict.fromkeys(value for value in hashtags if value))[:5]
    return MarketingMetadata(
        title=title,
        description=description,
        hashtags=hashtags or ["shorts", "insight", "creator"],
        hook=hook,
    )


def _fallback(transcript: str, source_title: str = "") -> MarketingMetadata:
    topic = _topic(source_title or transcript)
    title = _fallback_title(transcript, topic)
    return MarketingMetadata(
        title=title,
        description=(
            f"This clip examines {topic.lower()} and distills the speaker's central takeaway "
            "into one focused moment."
        ),
        hashtags=["shorts", "insight", "mindset"],
        hook=_fallback_hook(title, transcript, topic),
    )


def _valid_title(value: str, transcript: str, context: str = "") -> bool:
    words = value.split()
    lowered = value.lower()
    return (
        5 <= len(words) <= 10
        and len(value) <= 80
        and not _generic(value)
        and lowered not in _clean(transcript, 10_000).lower()
        and not _similar(value, _opening(transcript), threshold=0.65)
        and not _incomplete(value)
        and not _has_excessive_caps(value)
        and (not context or _context_score(value, context) > 0)
    )


def _valid_hook(
    value: str,
    title: str,
    transcript: str,
    context: str = "",
) -> bool:
    words = value.split()
    return (
        3 <= len(words) <= 6
        and len(value) <= 48
        and not _generic(value)
        and not _similar(value, title, threshold=0.5)
        and not _similar(value, _opening(transcript), threshold=0.55)
        and value.lower() not in transcript.lower()
        and not _incomplete(value)
        and not _has_excessive_caps(value)
        and (not context or _context_score(value, context) > 0)
    )


def _fallback_title(transcript: str, context: str = "") -> str:
    topic = _topic(context or transcript).split()[0]
    return f"The Simple Truth About {topic}"[:80]


def _fallback_hook(title: str, transcript: str, context: str = "") -> str:
    topic = _topic(context or transcript).split()[0]
    options = [
        f"Why {topic} Actually Matters",
        "The Part Most People Miss",
        "This Changes The Whole Point",
        "Here Is What Actually Matters",
    ]
    return next(
        value
        for value in options
        if not _similar(value, title, threshold=0.5)
        and not _similar(value, _opening(transcript), threshold=0.55)
    )


def _context_score(value: str, context: str) -> int:
    ignored = {
        "about", "actually", "before", "from", "into", "most", "people",
        "simple", "that", "the", "this", "what", "when", "why", "with",
    }
    value_words = {
        word
        for word in re.findall(r"[a-z0-9]+", value.lower())
        if len(word) >= 4 and word not in ignored
    }
    context_words = {
        word
        for word in re.findall(r"[a-z0-9]+", context.lower())
        if len(word) >= 4 and word not in ignored
    }
    return len(value_words & context_words)


def _clean(value: str, maximum: int) -> str:
    value = re.sub(r"[*_`#]+", "", value)
    value = re.sub(r"^(title|hook|description)\s*:\s*", "", value, flags=re.IGNORECASE)
    value = value.strip(" \t\r\n\"'")
    value = re.sub(
        r"\b([A-Za-z]+(?:['’][A-Za-z]+)?)(?:\s+\1\b)+",
        r"\1",
        value,
        flags=re.IGNORECASE,
    )
    return " ".join(value.split())[:maximum].rstrip(" ,;:-")


def _similar(left: str, right: str, threshold: float = 0.7) -> bool:
    def tokens(value: str) -> set[str]:
        return set(re.findall(r"[a-z0-9]+", value.lower()))

    left_words, right_words = tokens(left), tokens(right)
    if not left_words or not right_words:
        return False
    return len(left_words & right_words) / min(len(left_words), len(right_words)) >= threshold


def _generic(value: str) -> bool:
    lowered = value.lower()
    phrases = {
        "a powerful perspective",
        "think differently",
        "you won't believe",
        "watch until the end",
        "this is important",
        "must watch",
        "game changer",
    }
    return any(phrase in lowered for phrase in phrases)


def _incomplete(value: str) -> bool:
    final = re.sub(r"[^a-z']", "", value.lower().split()[-1])
    return final in {
        "a", "an", "and", "are", "as", "at", "but", "for", "from", "if",
        "in", "is", "it's", "of", "on", "or", "that", "the", "to", "with",
    }


def _has_excessive_caps(value: str) -> bool:
    letters = [character for character in value if character.isalpha()]
    uppercase = sum(character.isupper() for character in letters)
    return len(letters) >= 6 and uppercase / len(letters) > 0.65


def _opening(text: str) -> str:
    return " ".join(text.split()[:12])


def _topic(text: str) -> str:
    stopwords = {
        "about", "after", "again", "because", "could", "from", "have", "into",
        "just", "like", "really", "that", "their", "there", "these", "they",
        "this", "those", "very", "what", "when", "where", "which", "with",
        "would", "your", "youre", "it's", "its",
    }
    candidates = [
        word
        for word in re.findall(r"[A-Za-z][A-Za-z'-]+", text)
        if len(word) >= 4 and word.lower() not in stopwords
    ]
    unique = list(dict.fromkeys(word.lower() for word in candidates))
    return " ".join(word.title() for word in unique[:2]) or "This Idea"
