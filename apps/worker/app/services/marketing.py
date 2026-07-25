import json
import re
from urllib import error, request

from app.schemas import MarketingMetadata, TranscriptSegment


class MarketingGenerator:
    def __init__(self, ollama_host: str, model: str) -> None:
        self.endpoint = ollama_host.rstrip("/") + "/api/chat"
        self.model = model

    def generate(self, transcript: list[TranscriptSegment]) -> MarketingMetadata:
        text = " ".join(segment.text for segment in transcript)
        schema = MarketingMetadata.model_json_schema()
        payload = json.dumps(
            {
                "model": self.model,
                "stream": False,
                "format": schema,
                "messages": [
                    {
                        "role": "system",
                        "content": (
                            "Create four distinct marketing fields for this short clip. "
                            "TITLE: a specific, natural 5-10 word publishing title. "
                            "DESCRIPTION: 1-2 complete sentences that explain the clip's value "
                            "without copying the title or transcript. "
                            "HOOK: a truthful 3-7 word opening-card headline that creates "
                            "curiosity. It must not quote the dialogue, repeat the title, end as "
                            "an incomplete sentence, or contain Markdown. "
                            "HASHTAGS: 3-5 relevant lowercase tags without # symbols. "
                            "Never repeat a phrase between fields. Return JSON only."
                        ),
                    },
                    {"role": "user", "content": text[:6000]},
                ],
                "options": {"temperature": 0.4, "num_ctx": 2048, "num_predict": 384},
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
            metadata = MarketingMetadata.model_validate_json(content)
            return _normalize(metadata, text)
        except (error.URLError, TimeoutError, KeyError, ValueError, json.JSONDecodeError):
            topic = _topic(text)
            return _normalize(
                MarketingMetadata(
                    title=f"A Powerful Perspective on {topic}",
                    description=(
                        f"This clip explores {topic.lower()} and the practical idea behind it. "
                        "Watch the full moment for the key takeaway."
                    ),
                    hashtags=["shorts", "insight", "mindset", "creator"],
                    hook=f"Think Differently About {topic}",
                ),
                text,
            )


def _normalize(metadata: MarketingMetadata, transcript: str) -> MarketingMetadata:
    title = _clean(metadata.title, 100) or f"A Powerful Perspective on {_topic(transcript)}"
    hook = _clean(metadata.hook, 60)
    description = _clean(metadata.description, 300)
    topic = _topic(transcript)

    if not 3 <= len(hook.split()) <= 7 or _similar(hook, title) or _similar(
        hook, _opening(transcript)
    ):
        hook = f"Think Differently About {topic}"
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


def _similar(left: str, right: str) -> bool:
    def tokens(value: str) -> set[str]:
        return set(re.findall(r"[a-z0-9]+", value.lower()))

    left_words, right_words = tokens(left), tokens(right)
    if not left_words or not right_words:
        return False
    return len(left_words & right_words) / min(len(left_words), len(right_words)) >= 0.7


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
