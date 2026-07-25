from unittest.mock import Mock

from redis.exceptions import TimeoutError as RedisTimeoutError

from app.render_store import RenderJobStore


def test_render_wait_uses_reliable_processing_queue() -> None:
    client = Mock()
    client.brpoplpush.return_value = b"render-id"
    store = RenderJobStore(client, "renders")

    assert store.wait(timeout=1) == "render-id"
    client.brpoplpush.assert_called_once_with("renders", "renders:processing", timeout=1)


def test_render_wait_treats_timeout_as_idle() -> None:
    client = Mock()
    client.brpoplpush.side_effect = RedisTimeoutError("idle")

    assert RenderJobStore(client, "renders").wait() is None


def test_render_recovery_and_acknowledgement() -> None:
    client = Mock()
    client.rpoplpush.side_effect = [b"one", b"two", None]
    store = RenderJobStore(client, "renders")

    assert store.recover_interrupted() == 2
    store.acknowledge("one")
    client.lrem.assert_called_once_with("renders:processing", 1, "one")


def test_render_release_returns_unstarted_job_to_queue() -> None:
    client = Mock()
    pipe = client.pipeline.return_value
    store = RenderJobStore(client, "renders")

    store.release("render-id")

    pipe.lrem.assert_called_once_with("renders:processing", 1, "render-id")
    pipe.rpush.assert_called_once_with("renders", "render-id")
    pipe.execute.assert_called_once()
