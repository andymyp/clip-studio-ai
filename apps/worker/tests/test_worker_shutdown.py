from types import SimpleNamespace
from unittest.mock import patch

from app import worker


def test_second_stop_requests_forced_shutdown() -> None:
    worker.shutdown.clear()
    worker.force_shutdown.clear()
    with patch.object(worker.logger, "info") as info, patch.object(
        worker.logger, "warning"
    ) as warning:
        worker.stop(15, None)
        try:
            worker.stop(15, None)
        except worker.ForcedShutdown:
            pass

    assert worker.shutdown.is_set()
    assert worker.force_shutdown.is_set()
    info.assert_called_once()
    warning.assert_called_once()
    worker.shutdown.clear()
    worker.force_shutdown.clear()


def test_third_stop_does_not_interrupt_forced_shutdown_cleanup() -> None:
    worker.shutdown.set()
    worker.force_shutdown.set()

    with patch.object(worker.logger, "warning") as warning:
        worker.stop(15, None)

    warning.assert_called_once_with("Forced shutdown is already in progress")
    worker.shutdown.clear()
    worker.force_shutdown.clear()


def test_forced_shutdown_releases_claimed_job() -> None:
    store = SimpleNamespace()

    def interrupt_acknowledge(_job_id: str) -> None:
        raise worker.ForcedShutdown

    store.acknowledge = interrupt_acknowledge
    released: list[str] = []
    store.release = released.append
    processor = SimpleNamespace(store=store, process=lambda _job_id: None)

    try:
        worker._run_claimed_job(processor, "job-1")
    except worker.ForcedShutdown:
        pass

    assert released == ["job-1"]
    worker.force_shutdown.clear()
