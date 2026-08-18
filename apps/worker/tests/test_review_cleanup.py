import json
from pathlib import Path
from unittest.mock import Mock

from app.services.review_cleanup import ReviewFileSweeper

ORPHAN_ID = "11111111-1111-4111-8111-111111111111"
ACTIVE_ID = "22222222-2222-4222-8222-222222222222"


def _review_files(storage: Path, job_id: str) -> tuple[Path, Path]:
    clip = storage / "output" / "clips" / job_id / "clip.mp4"
    transcript = storage / "output" / "transcripts" / f"{job_id}.json"
    clip.parent.mkdir(parents=True)
    transcript.parent.mkdir(parents=True, exist_ok=True)
    clip.write_bytes(b"clip")
    transcript.write_text("[]", encoding="utf-8")
    return clip, transcript


def test_sweeper_deletes_files_after_analysis_redis_key_expires(tmp_path: Path) -> None:
    clip, transcript = _review_files(tmp_path, ORPHAN_ID)
    client = Mock()
    client.lrange.return_value = []
    client.scan_iter.return_value = []
    client.exists.return_value = 0

    removed = ReviewFileSweeper(client, tmp_path, "analysis").sweep()

    assert removed == 1
    assert not clip.parent.exists()
    assert not transcript.exists()


def test_sweeper_keeps_files_for_existing_analysis_job(tmp_path: Path) -> None:
    clip, transcript = _review_files(tmp_path, ACTIVE_ID)
    client = Mock()
    client.lrange.return_value = []
    client.scan_iter.return_value = []
    client.exists.return_value = 1

    removed = ReviewFileSweeper(client, tmp_path, "analysis").sweep()

    assert removed == 0
    assert clip.exists()
    assert transcript.exists()


def test_sweeper_keeps_source_needed_by_failed_render_retry(tmp_path: Path) -> None:
    clip, transcript = _review_files(tmp_path, ACTIVE_ID)
    client = Mock()
    client.lrange.return_value = []
    client.exists.return_value = 0
    client.scan_iter.return_value = [b"clipstudio:render:job:render-id"]
    client.get.return_value = json.dumps(
        {
            "status": "failed",
            "analysis_job_id": ACTIVE_ID,
        }
    )

    removed = ReviewFileSweeper(client, tmp_path, "analysis").sweep()

    assert removed == 0
    assert clip.exists()
    assert transcript.exists()


def test_sweeper_deletes_orphan_transcript_without_clip_directory(tmp_path: Path) -> None:
    transcript = tmp_path / "output" / "transcripts" / f"{ORPHAN_ID}.json"
    transcript.parent.mkdir(parents=True)
    transcript.write_text("[]", encoding="utf-8")
    client = Mock()
    client.lrange.return_value = []
    client.scan_iter.return_value = []
    client.exists.return_value = 0

    removed = ReviewFileSweeper(client, tmp_path, "analysis").sweep()

    assert removed == 1
    assert not transcript.exists()
