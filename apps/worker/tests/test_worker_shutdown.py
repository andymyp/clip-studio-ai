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
