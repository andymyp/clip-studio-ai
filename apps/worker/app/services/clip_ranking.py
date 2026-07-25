import json
import math
from urllib import error, request

from app.schemas import RankedClip, TranscriptSegment


class ClipRankingError(RuntimeError):
    pass


class ClipRankingService:
    def __init__(
        self,
        ollama_host: str,
        model: str = "qwen2.5:7b",
        minimum_duration: float = 20,
        maximum_duration: float = 60,
        maximum_candidates: int = 8,
        window_duration: float = 180,
        window_overlap: float = 20,
        maximum_analysis_windows: int = 1,
    ) -> None:
        self.endpoint = ollama_host.rstrip("/") + "/api/chat"
        self.model = model
        self.minimum_duration = minimum_duration
        self.maximum_duration = maximum_duration
        self.maximum_candidates = maximum_candidates
        self.window_duration = window_duration
        self.window_overlap = window_overlap
        self.maximum_analysis_windows = maximum_analysis_windows

    def analyze(self, transcript: list[TranscriptSegment]) -> list[RankedClip]:
        if not transcript:
            return []
        ranked: list[RankedClip] = []
        windows = self._select_windows(self._windows(transcript))
        for window in windows:
            try:
                ranked.extend(self._analyze_window(window))
            except ClipRankingError:
                continue
        if not ranked:
            ranked = self._heuristic_candidates(transcript)

        normalized = [self._normalize(clip, transcript[-1].end) for clip in ranked]
        valid = [clip for clip in normalized if clip is not None]
        deduplicated: list[RankedClip] = []
        for clip in sorted(valid, key=lambda candidate: candidate.score, reverse=True):
            if any(
                max(clip.start, existing.start) < min(clip.end, existing.end)
                for existing in deduplicated
            ):
                continue
            deduplicated.append(clip)
            if len(deduplicated) == self.maximum_candidates:
                break
        return deduplicated

    def _heuristic_candidates(
        self,
        transcript: list[TranscriptSegment],
    ) -> list[RankedClip]:
        target_duration = min(45, self.maximum_duration)
        candidates: list[tuple[float, RankedClip]] = []
        for index, start_segment in enumerate(transcript):
            target_end = start_segment.start + target_duration
            end_index = index
            while end_index + 1 < len(transcript) and transcript[end_index].end < target_end:
                end_index += 1
            end_segment = transcript[end_index]
            duration = end_segment.end - start_segment.start
            if duration < self.minimum_duration or duration > self.maximum_duration:
                continue
            text = " ".join(
                segment.text for segment in transcript[index : end_index + 1]
            )
            words = text.split()
            if not words:
                continue
            hook_count = sum(
                text.lower().count(term)
                for term in ("why", "how", "never", "best", "important", "problem")
            )
            density = len({word.lower().strip(".,!?") for word in words}) / len(words)
            raw_score = len(words) / max(duration, 1) + hook_count * 2 + density * 5
            score = min(89, 68 + raw_score)
            candidates.append(
                (
                    raw_score,
                    RankedClip(
                        start=start_segment.start,
                        end=end_segment.end,
                        score=score,
                        reason="Strong information density and standalone hook",
                    ),
                )
            )

        selected: list[RankedClip] = []
        for _, candidate in sorted(candidates, key=lambda item: item[0], reverse=True):
            if any(
                max(candidate.start, existing.start) < min(candidate.end, existing.end)
                for existing in selected
            ):
                continue
            selected.append(candidate)
            if len(selected) == self.maximum_candidates:
                break
        return selected

    def _select_windows(
        self,
        windows: list[list[TranscriptSegment]],
    ) -> list[list[TranscriptSegment]]:
        if len(windows) <= self.maximum_analysis_windows:
            return windows
        hook_terms = {
            "why",
            "how",
            "secret",
            "mistake",
            "never",
            "best",
            "important",
            "imagine",
            "surprising",
            "problem",
        }

        def score(window: list[TranscriptSegment]) -> float:
            text = " ".join(segment.text.lower() for segment in window)
            words = text.split()
            hooks = sum(text.count(term) for term in hook_terms)
            questions = text.count("?")
            return len(words) + hooks * 20 + questions * 12

        selected = sorted(windows, key=score, reverse=True)[: self.maximum_analysis_windows]
        return sorted(selected, key=lambda window: window[0].start)

    def _analyze_window(self, transcript: list[TranscriptSegment]) -> list[RankedClip]:
        schema = {
            "type": "object",
            "properties": {
                "clips": {
                    "type": "array",
                    "items": {
                        "type": "object",
                        "properties": {
                            "start": {"type": "number"},
                            "end": {"type": "number"},
                            "score": {"type": "number", "minimum": 0, "maximum": 100},
                            "reason": {"type": "string"},
                        },
                        "required": ["start", "end", "score", "reason"],
                    },
                },
            },
            "required": ["clips"],
        }
        per_window = min(3, self.maximum_candidates)
        prompt = (
            "Find the best standalone moments for vertical short-form video. "
            f"Return at most {per_window} moments, each between "
            f"{self.minimum_duration:g} and {self.maximum_duration:g} seconds. "
            "Score hook strength, emotion, curiosity, replay probability, and "
            "information density. Do not invent timestamps. Prefer complete thoughts. "
            "The timestamps are absolute seconds from the source video. "
            "Transcript JSON:\n"
            + json.dumps([segment.model_dump() for segment in transcript], ensure_ascii=False)
        )
        payload = json.dumps(
            {
                "model": self.model,
                "stream": False,
                "format": schema,
                "messages": [
                    {
                        "role": "system",
                        "content": "You are a precise short-video clip editor. Return JSON only.",
                    },
                    {"role": "user", "content": prompt},
                ],
                "options": {
                    "temperature": 0.2,
                    "num_ctx": 4096,
                    "num_predict": 384,
                },
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
                body = json.load(response)
        except (error.URLError, TimeoutError, json.JSONDecodeError) as exc:
            raise ClipRankingError(f"Ollama request failed: {exc}") from exc
        try:
            result = json.loads(body["message"]["content"])
            candidates = result["clips"]
            ranked = [RankedClip.model_validate(candidate) for candidate in candidates]
        except (KeyError, TypeError, json.JSONDecodeError, ValueError) as exc:
            raise ClipRankingError("Ollama returned invalid clip ranking JSON") from exc
        return ranked

    def _windows(
        self,
        transcript: list[TranscriptSegment],
    ) -> list[list[TranscriptSegment]]:
        start, end = transcript[0].start, transcript[-1].end
        step = self.window_duration - self.window_overlap
        count = max(1, math.ceil((end - start - self.window_overlap) / step))
        windows: list[list[TranscriptSegment]] = []
        for index in range(count):
            window_start = start + index * step
            window_end = window_start + self.window_duration
            window = [
                segment
                for segment in transcript
                if segment.end >= window_start and segment.start <= window_end
            ]
            if window:
                windows.append(window)
        return windows

    def _normalize(self, clip: RankedClip, transcript_end: float) -> RankedClip | None:
        start = max(0, min(clip.start, transcript_end))
        end = max(start, min(clip.end, transcript_end))
        if end - start < self.minimum_duration:
            end = min(transcript_end, start + self.minimum_duration)
            if end - start < self.minimum_duration:
                start = max(0, end - self.minimum_duration)
        if end - start > self.maximum_duration:
            end = start + self.maximum_duration
        if end <= start or end > transcript_end + 1:
            return None
        return clip.model_copy(update={"start": start, "end": end})
