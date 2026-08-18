import json
import shutil
from pathlib import Path
from uuid import UUID

from redis import Redis


class ReviewFileSweeper:
    """Deletes analysis media after its Redis job has expired."""

    def __init__(
        self,
        client: Redis,
        storage: Path,
        analysis_queue: str,
    ) -> None:
        self.client = client
        self.storage = storage.resolve()
        self.analysis_queue = analysis_queue
        self.analysis_processing_queue = f"{analysis_queue}:processing"

    def sweep(self) -> int:
        clips_root = self.storage / "output" / "clips"
        protected = self._queued_analysis_ids() | self._active_render_analysis_ids()
        expired: set[str] = set()
        removed = 0
        directories = clips_root.iterdir() if clips_root.is_dir() else ()
        for directory in directories:
            if not directory.is_dir() or not _is_uuid(directory.name):
                continue
            job_id = directory.name
            if job_id in protected or self.client.exists(_analysis_key(job_id)):
                continue
            expired.add(job_id)

            resolved = directory.resolve()
            if resolved.parent != clips_root.resolve():
                continue
            shutil.rmtree(resolved)
            transcript = self.storage / "output" / "transcripts" / f"{job_id}.json"
            transcript.unlink(missing_ok=True)
            removed += 1

        transcripts_root = self.storage / "output" / "transcripts"
        if transcripts_root.is_dir():
            for transcript in transcripts_root.glob("*.json"):
                job_id = transcript.stem
                if not _is_uuid(job_id) or job_id in protected:
                    continue
                if job_id not in expired and self.client.exists(_analysis_key(job_id)):
                    continue
                transcript.unlink(missing_ok=True)
                if job_id not in expired:
                    removed += 1
        return removed

    def _queued_analysis_ids(self) -> set[str]:
        values = [
            *self.client.lrange(self.analysis_queue, 0, -1),
            *self.client.lrange(self.analysis_processing_queue, 0, -1),
        ]
        return {_decode(value) for value in values}

    def _active_render_analysis_ids(self) -> set[str]:
        protected: set[str] = set()
        for key in self.client.scan_iter(match="clipstudio:render:job:*"):
            payload = self.client.get(key)
            if not payload:
                continue
            try:
                state = json.loads(_decode(payload))
            except (TypeError, json.JSONDecodeError):
                continue
            if state.get("status") == "completed":
                continue
            analysis_job_id = state.get("analysis_job_id")
            if isinstance(analysis_job_id, str):
                protected.add(analysis_job_id)
        return protected


def _analysis_key(job_id: str) -> str:
    return f"clipstudio:analysis:job:{job_id}"


def _decode(value: object) -> str:
    return value.decode() if isinstance(value, bytes) else str(value)


def _is_uuid(value: str) -> bool:
    try:
        UUID(value)
    except ValueError:
        return False
    return True
