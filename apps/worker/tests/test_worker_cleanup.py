from pathlib import Path

from app.worker import AnalysisJobProcessor


def test_failed_job_cleanup_removes_transcript_and_clip_directory(tmp_path: Path) -> None:
    job_id = "job-id"
    transcript = tmp_path / "output" / "transcripts" / f"{job_id}.json"
    clip = tmp_path / "output" / "clips" / job_id / "clip.mp4"
    transcript.parent.mkdir(parents=True)
    clip.parent.mkdir(parents=True)
    transcript.write_text("[]", encoding="utf-8")
    clip.write_bytes(b"partial")

    processor = AnalysisJobProcessor.__new__(AnalysisJobProcessor)
    processor.storage = tmp_path
    processor._cleanup(job_id)

    assert not transcript.exists()
    assert not clip.parent.exists()
