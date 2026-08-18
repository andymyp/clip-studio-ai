import json
from unittest.mock import Mock

from redis.exceptions import TimeoutError as RedisTimeoutError

from app.job_store import COMPLETED_REVIEW_TTL_SECONDS, AnalysisJobStore
from app.schemas import AnalysisJob, AnalyzeVideoRequest


def test_wait_treats_idle_socket_timeout_as_empty_queue() -> None:
    client = Mock()
    client.brpoplpush.side_effect = RedisTimeoutError("Timeout reading from socket")
    store = AnalysisJobStore(client, "analysis")

    assert store.wait(timeout=5) is None


def test_wait_returns_queued_job_id() -> None:
    client = Mock()
    client.brpoplpush.return_value = b"job-id"
    store = AnalysisJobStore(client, "analysis")

    assert store.wait(timeout=5) == "job-id"


def test_recover_interrupted_jobs() -> None:
    client = Mock()
    client.rpoplpush.side_effect = [b"job-1", b"job-2", None]
    client.scan_iter.return_value = []
    store = AnalysisJobStore(client, "analysis")

    assert store.recover_interrupted() == 2


def test_recover_legacy_orphaned_processing_job() -> None:
    client = Mock()
    client.rpoplpush.return_value = None
    job = {
        "id": "job-id",
        "user_id": "user-id",
        "external_id": "video-id",
        "url": "https://youtube.com/watch?v=test",
        "title": "Test",
        "platform": "youtube",
        "status": "processing",
        "progress": 45,
        "message": "Downloading clip section 1 of 8",
    }
    client.scan_iter.return_value = [b"clipstudio:analysis:job:job-id"]
    client.get.return_value = json.dumps(job)
    client.lpos.return_value = None
    store = AnalysisJobStore(client, "analysis")

    assert store.recover_interrupted() == 1
    client.lpush.assert_called_once_with("analysis", "job-id")


def test_create_reuses_cached_successful_review() -> None:
    client = Mock()
    cached = {
        "id": "cached-job",
        "user_id": "user-id",
        "external_id": "video-id",
        "url": "https://youtube.com/watch?v=video-id",
        "title": "Test",
        "platform": "youtube",
        "status": "completed",
        "progress": 100,
        "message": "Generated clips",
    }
    client.get.side_effect = [b"cached-job", json.dumps(cached)]
    store = AnalysisJobStore(client, "analysis")
    payload = AnalyzeVideoRequest(
        user_id="user-id",
        external_id="video-id",
        url="https://youtube.com/watch?v=video-id",
        title="Test",
        platform="youtube",
    )

    result = store.create(payload)

    assert result.id == "cached-job"
    client.lpush.assert_not_called()


def test_completed_review_and_job_expire_after_thirty_minutes() -> None:
    client = Mock()
    store = AnalysisJobStore(client, "analysis")
    job = AnalysisJob(
        id="job-id",
        user_id="user-id",
        external_id="video-id",
        url="https://youtube.com/watch?v=video-id",
        title="Test",
        platform="youtube",
        status="completed",
        progress=100,
        message="Generated clips",
    )

    store.save(job)

    assert COMPLETED_REVIEW_TTL_SECONDS == 30 * 60
    assert client.set.call_count == 2
    assert all(
        call.kwargs["ex"] == COMPLETED_REVIEW_TTL_SECONDS
        for call in client.set.call_args_list
    )
