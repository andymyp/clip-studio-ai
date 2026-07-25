from unittest.mock import patch

from app import worker


def test_stop_is_idempotent_and_requests_graceful_shutdown() -> None:
    worker.shutdown.clear()
    with patch.object(worker.logger, "info") as info, patch.object(
        worker.logger, "warning"
    ) as warning:
        worker.stop(15, None)
        worker.stop(15, None)

    assert worker.shutdown.is_set()
    info.assert_called_once()
    warning.assert_called_once()
    worker.shutdown.clear()
