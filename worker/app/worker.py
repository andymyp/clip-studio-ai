import logging
import signal
import threading

from redis import Redis

from app.config import get_settings

logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)
shutdown = threading.Event()


def stop(_signum: int, _frame: object) -> None:
    shutdown.set()


def run() -> None:
    settings = get_settings()
    client = Redis.from_url(settings.redis_url, decode_responses=True)
    client.ping()
    logger.info("Worker foundation connected and awaiting future job contracts")

    while not shutdown.wait(timeout=5):
        client.ping()


if __name__ == "__main__":
    signal.signal(signal.SIGTERM, stop)
    signal.signal(signal.SIGINT, stop)
    run()
