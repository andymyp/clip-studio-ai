import logging
import shutil
import signal
import subprocess
import threading
from pathlib import Path
from uuid import uuid4

from redis import Redis
from redis.exceptions import RedisError

from .config import get_settings
from .job_store import AnalysisJobStore
from .schemas import GeneratedClip
from .services import ClipRankingService, PartialDownloaderService, TranscriptService
from .services.partial_downloader import PartialDownloadError

logging.basicConfig(
    level=logging.INFO,
    format="%(asctime)s %(levelname)s %(name)s %(message)s",
)
logger = logging.getLogger(__name__)
shutdown = threading.Event()


def stop(_signum: int, _frame: object) -> None:
    shutdown.set()


class AnalysisJobProcessor:
    def __init__(self) -> None:
        settings = get_settings()
        client = Redis.from_url(settings.redis_url)
        self.store = AnalysisJobStore(client, settings.job_queue_name)
        self.transcripts = TranscriptService(
            settings.transcript_language,
            js_runtime=settings.ytdlp_js_runtime,
            impersonate=settings.ytdlp_impersonate,
            cookie_file=settings.ytdlp_cookie_file,
            sleep_requests=settings.ytdlp_sleep_requests,
            sleep_subtitles=settings.ytdlp_sleep_subtitles,
        )
        self.ranking = ClipRankingService(
            settings.ollama_host,
            settings.ollama_model,
            settings.clip_min_duration,
            settings.clip_max_duration,
            settings.clip_max_candidates,
        )
        self.downloader = PartialDownloaderService(
            settings.storage_path,
            js_runtime=settings.ytdlp_js_runtime,
            impersonate=settings.ytdlp_impersonate,
            cookie_file=settings.ytdlp_cookie_file,
            sleep_requests=settings.ytdlp_sleep_requests,
        )
        self.storage = Path(settings.storage_path).resolve()

    def process(self, job_id: str) -> None:
        job = self.store.get(job_id)
        if job is None:
            logger.warning("Ignoring missing job %s", job_id)
            return
        try:
            job.status = "processing"
            job.progress = 10
            job.message = "Extracting YouTube subtitles"
            self.store.save(job)
            transcript = self.transcripts.extract(job.url)

            transcript_directory = self.storage / "output" / "transcripts"
            transcript_directory.mkdir(parents=True, exist_ok=True)
            transcript_path = transcript_directory / f"{job.id}.json"
            transcript_path.write_text(
                self.transcripts.to_json(transcript),
                encoding="utf-8",
            )

            job.progress = 35
            job.message = "Ranking the best short-video moments"
            self.store.save(job)
            ranked = self.ranking.analyze(transcript)
            if not ranked:
                raise RuntimeError("AI analyzer returned no valid clip moments")

            clips: list[GeneratedClip] = []
            download_errors: list[str] = []
            for index, candidate in enumerate(ranked):
                job.progress = 45 + (index / len(ranked)) * 45
                job.message = f"Downloading clip section {index + 1} of {len(ranked)}"
                self.store.save(job)
                clip_id = str(uuid4())
                try:
                    output = self.downloader.download(
                        job.url,
                        candidate.start,
                        candidate.end,
                        job.id,
                        clip_id,
                    )
                except (PartialDownloadError, subprocess.TimeoutExpired, OSError) as exc:
                    logger.warning(
                        "Skipping failed clip section %s for job %s: %s",
                        index + 1,
                        job.id,
                        exc,
                    )
                    download_errors.append(str(exc))
                    continue
                clips.append(
                    GeneratedClip(
                        id=clip_id,
                        start=candidate.start,
                        end=candidate.end,
                        score=candidate.score,
                        reason=candidate.reason,
                        output_path=str(output.relative_to(self.storage)),
                        media_url=f"/media/clips/{job.id}/{output.name}",
                    )
                )

            if not clips:
                detail = download_errors[0] if download_errors else "no clip sections succeeded"
                raise RuntimeError(f"All partial clip downloads failed: {detail}")
            job.clips = clips
            job.status = "completed"
            job.progress = 100
            job.message = f"Generated {len(clips)} clip candidates"
            self.store.save(job)
        except Exception as exc:
            logger.exception("Analysis job %s failed", job.id)
            self._cleanup(job.id)
            self.store.invalidate_review(job)
            job.clips = []
            job.status = "failed"
            job.message = "Analysis failed"
            job.error = str(exc)
            self.store.save(job)

    def _cleanup(self, job_id: str) -> None:
        transcript = self.storage / "output" / "transcripts" / f"{job_id}.json"
        clip_directory = self.storage / "output" / "clips" / job_id
        try:
            transcript.unlink(missing_ok=True)
        except OSError:
            logger.exception("Could not delete failed transcript %s", transcript)
        try:
            if clip_directory.is_dir():
                shutil.rmtree(clip_directory)
        except OSError:
            logger.exception("Could not delete failed clip directory %s", clip_directory)


def run() -> None:
    processor = AnalysisJobProcessor()
    processor.store.client.ping()
    recovered = processor.store.recover_interrupted()
    if recovered:
        logger.warning("Recovered %s interrupted analysis job(s)", recovered)
    logger.info("Analysis worker is ready")
    while not shutdown.is_set():
        try:
            job_id = processor.store.wait(timeout=5)
            if job_id:
                try:
                    processor.process(job_id)
                finally:
                    processor.store.acknowledge(job_id)
        except RedisError:
            logger.exception("Redis unavailable; retrying in 2 seconds")
            shutdown.wait(2)


if __name__ == "__main__":
    signal.signal(signal.SIGTERM, stop)
    signal.signal(signal.SIGINT, stop)
    run()
