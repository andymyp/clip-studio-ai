import json
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
from .render_store import RenderJobStore
from .schemas import (
    EditInterval,
    GeneratedClip,
    OptimizationPlan,
    RenderJobState,
    TranscriptSegment,
)
from .services import (
    AudioMasteringService,
    ClipRankingService,
    MarketingGenerator,
    PartialDownloaderService,
    RenderQualityService,
    RenderService,
    RetentionEditService,
    SubjectReframeService,
    SubtitleGenerator,
    TranscriptService,
)
from .services.partial_downloader import PartialDownloadError

logging.basicConfig(
    level=logging.INFO,
    format="%(asctime)s %(levelname)s %(name)s %(message)s",
)
logger = logging.getLogger(__name__)
shutdown = threading.Event()
force_shutdown = threading.Event()


class ForcedShutdown(BaseException):
    """Interrupt an active job after a second termination signal."""


def stop(_signum: int, _frame: object) -> None:
    if shutdown.is_set():
        if force_shutdown.is_set():
            logger.warning("Forced shutdown is already in progress")
            return
        logger.warning("Second shutdown requested; interrupting and requeueing active work")
        force_shutdown.set()
        raise ForcedShutdown
    logger.info("Shutdown requested; stopping intake and finishing the active job")
    shutdown.set()


def install_signal_handlers() -> None:
    signal.signal(signal.SIGTERM, stop)
    signal.signal(signal.SIGINT, stop)
    if hasattr(signal, "SIGBREAK"):
        signal.signal(signal.SIGBREAK, stop)


def _run_claimed_job(processor: object, job_id: str) -> None:
    """Process and settle one claimed job without losing it on forced shutdown."""
    try:
        processor.process(job_id)  # type: ignore[attr-defined]
        if force_shutdown.is_set():
            raise ForcedShutdown
        processor.store.acknowledge(job_id)  # type: ignore[attr-defined]
    except ForcedShutdown:
        processor.store.release(job_id)  # type: ignore[attr-defined]
        raise


class AnalysisJobProcessor:
    def __init__(self) -> None:
        settings = get_settings()
        client = Redis.from_url(settings.redis_url)
        self.store = AnalysisJobStore(client, settings.job_queue_name)
        self.redis = client
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
        self.context_before = settings.clip_context_before
        self.context_after = settings.clip_context_after

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
            self.ranking.preferred_duration = self._learned_duration()
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
                section_start, section_end = _semantic_clip_bounds(
                    candidate.start,
                    candidate.end,
                    transcript,
                    self.context_before,
                    self.context_after,
                )
                try:
                    output = self.downloader.download(
                        job.url,
                        section_start,
                        section_end,
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
                        start=section_start,
                        end=section_end,
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

    def _learned_duration(self) -> float | None:
        outcomes = self.redis.lrange("clipstudio:ranking:outcomes", 0, 199)
        weighted_duration = 0.0
        weight = 0.0
        for raw in outcomes:
            try:
                item = json.loads(raw)
                score = max(0.0, float(item["viral_score"]))
                duration = float(item["duration"])
                swipe = max(0.0, min(100.0, float(item.get("swiped_away_percentage", 0))))
                engaged = max(0.0, min(100.0, float(item.get("engaged_rate", 0))))
                replay = max(0.0, min(100.0, float(item.get("replay_rate", 0))))
            except (KeyError, TypeError, ValueError, json.JSONDecodeError):
                continue
            score *= max(0.25, 1 - swipe / 125)
            score *= 1 + min(0.35, (engaged + replay) / 400)
            if (
                score < 50
                or not self.ranking.minimum_duration
                <= duration
                <= self.ranking.maximum_duration
            ):
                continue
            weighted_duration += duration * score
            weight += score
        return weighted_duration / weight if weight else None

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


class RenderJobProcessor:
    def __init__(self) -> None:
        settings = get_settings()
        client = Redis.from_url(settings.redis_url)
        self.store = RenderJobStore(client, settings.render_queue_name)
        self.storage = Path(settings.storage_path).resolve()
        self.subtitles = SubtitleGenerator()
        self.audio = AudioMasteringService()
        self.quality = RenderQualityService()
        self.renderer = RenderService()
        self.retention = RetentionEditService()
        self.reframe = SubjectReframeService()
        self.marketing = MarketingGenerator(settings.ollama_host, settings.ollama_model)
        self.context_before = settings.clip_context_before
        self.context_after = settings.clip_context_after

    def process(self, job_id: str) -> None:
        job = self.store.get(job_id)
        if job is None:
            return
        output_directory = self.storage / "output" / "rendered" / job.id
        try:
            if not job.rights_confirmed:
                raise RuntimeError("Reusable-content rights must be confirmed before rendering")
            job.status = "processing"
            self._progress(job, 10, "Preparing transcript")
            transcript_path = (
                self.storage / "output" / "transcripts" / f"{job.analysis_job_id}.json"
            )
            data = json.loads(transcript_path.read_text(encoding="utf-8"))
            transcript = [
                TranscriptSegment(
                    text=item["text"],
                    start=max(0, float(item["start"]) - job.start),
                    end=min(job.end - job.start, float(item["end"]) - job.start),
                )
                for item in data
                if float(item["end"]) > job.start and float(item["start"]) < job.end
            ]
            transcript = [segment for segment in transcript if segment.end > segment.start]
            if not transcript:
                raise RuntimeError("No transcript segments overlap the selected clip")

            clip = (self.storage / job.clip_path).resolve()
            if self.storage not in clip.parents:
                raise RuntimeError("Clip path is outside storage")
            duration = job.end - job.start
            self._progress(job, 20, "Removing dead air and filler")
            intro_silence = min(self.context_before, 1.35, max(0.85, duration * 0.06))
            outro_silence = min(self.context_after, 0.9, max(0.5, duration * 0.05))
            retention_intervals = self.retention.plan(
                clip,
                duration,
                transcript,
                preserve_start=intro_silence,
                preserve_end=outro_silence,
            )
            source_transcript = transcript
            transcript = self.retention.remap(source_transcript, retention_intervals)
            if not transcript:
                raise RuntimeError("Retention edit removed the entire transcript")
            base_edited_duration = sum(
                item.end - item.start for item in retention_intervals
            )
            playback_speed = _readability_speed(
                transcript,
                intro_silence,
                max(intro_silence, base_edited_duration - outro_silence),
            )
            timed_transcript = [
                segment.model_copy(
                    update={
                        "start": segment.start / playback_speed,
                        "end": segment.end / playback_speed,
                    }
                )
                for segment in transcript
            ]

            # Split long continuous sections into visual beats. Audio remains continuous,
            # while each beat can use an updated face center and subtle punch-in.
            visual_intervals = _visual_beats(retention_intervals, source_transcript)
            self._progress(job, 30, "Tracking the primary speaker")
            centers, layouts, zooms = self.reframe.plan(clip, visual_intervals)

            self._progress(job, 38, "Choosing the strongest hook and packaging")
            job.marketing = self.marketing.generate(transcript, job.source_title)
            attribution = " | ".join(
                value
                for value in (
                    job.source_username or "YouTube",
                    job.source_title,
                    job.source_url,
                )
                if value
            )
            job.marketing.description = (
                f"{job.marketing.description.rstrip()}\n\nSource: {attribution}"
            )
            important_phrases = [job.marketing.hook, job.marketing.title]
            edited_duration = base_edited_duration / playback_speed
            rendered_intro = intro_silence / playback_speed
            rendered_outro = outro_silence / playback_speed
            caption_end = max(rendered_intro, edited_duration - rendered_outro)
            caption_transcript = [
                segment
                for segment in timed_transcript
                if segment.start >= rendered_intro and segment.end <= caption_end
            ]
            output_directory.mkdir(parents=True, exist_ok=True)
            subtitle = self.subtitles.generate(
                caption_transcript,
                output_directory / "subtitle.ass",
                important_phrases=important_phrases,
                platform_profile=job.platform_profile,
            )
            job.subtitle_path = str(subtitle.relative_to(self.storage))
            job.optimization = OptimizationPlan(
                intervals=visual_intervals,
                face_centers=centers,
                layouts=layouts,
                zooms=zooms,
                removed_seconds=round(
                    duration - sum(item.end - item.start for item in retention_intervals), 2
                ),
                hook=job.marketing.hook,
                pattern_interrupts=[
                    round(item.start, 2) for item in visual_intervals[1:]
                ],
                important_phrases=[value for value in important_phrases if value],
                playback_speed=playback_speed,
                platform_profile=job.platform_profile,
            )
            self._progress(job, 48, "Dynamic Shorts captions generated")
            self._progress(job, 52, "Measuring and mastering speech audio")
            audio_measurement = self.audio.measure(clip)
            if audio_measurement:
                job.optimization.audio_loudness_lufs = audio_measurement["input_i"]
            final = self.renderer.render(
                clip,
                subtitle,
                output_directory / "final.mp4",
                job.watermark_text,
                job.source_url,
                lambda value, message: self._progress(job, value, message),
                visual_intervals,
                centers,
                job.marketing.hook,
                job.source_username,
                intro_silence,
                outro_silence,
                playback_speed,
                audio_measurement,
                job.platform_profile,
                layouts,
                zooms,
            )
            self._progress(job, 92, "Validating render quality")
            job.quality = self.quality.inspect(final, edited_duration)
            self._progress(job, 96, "Finalizing optimized metadata")
            job.output_path = str(final.relative_to(self.storage))
            job.media_url = f"/media/rendered/{job.id}/final.mp4"
            job.status = "completed"
            self._progress(job, 100, "Render completed")
        except Exception as exc:
            logger.exception("Render job %s failed", job.id)
            if output_directory.is_dir():
                shutil.rmtree(output_directory, ignore_errors=True)
            job.status = "failed"
            job.error = str(exc)
            job.message = "Render failed"
            self.store.save(job)

    def _progress(self, job: RenderJobState, value: float, message: str) -> None:
        job.progress = value
        job.message = message
        self.store.save(job)


def _visual_beats(
    intervals: list[EditInterval],
    transcript: list[TranscriptSegment] | None = None,
    minimum_seconds: float = 4.0,
    maximum_seconds: float = 9.0,
) -> list[EditInterval]:
    beats: list[EditInterval] = []
    for interval in intervals:
        cursor = interval.start
        sentence_ends = [
            segment.end
            for segment in transcript or []
            if cursor + minimum_seconds <= segment.end < interval.end
            and segment.text.rstrip().endswith((".", "!", "?"))
        ]
        while interval.end - cursor > maximum_seconds:
            choices = [
                value
                for value in sentence_ends
                if cursor + minimum_seconds <= value <= cursor + maximum_seconds
            ]
            boundary = choices[-1] if choices else min(interval.end, cursor + maximum_seconds)
            beats.append(EditInterval(start=cursor, end=boundary))
            cursor = boundary
        if interval.end - cursor >= 0.25:
            beats.append(EditInterval(start=cursor, end=interval.end))
    return beats


def _semantic_clip_bounds(
    start: float,
    end: float,
    transcript: list[TranscriptSegment],
    context_before: float,
    context_after: float,
) -> tuple[float, float]:
    overlapping = [
        segment for segment in transcript if segment.end > start and segment.start < end
    ]
    core_start = overlapping[0].start if overlapping else start
    core_end = overlapping[-1].end if overlapping else end
    for segment in transcript:
        if segment.end < end:
            continue
        core_end = segment.end
        if segment.text.rstrip().endswith((".", "!", "?")) or segment.end >= end + 4:
            break
    return max(0, core_start - context_before), core_end + context_after


def _readability_speed(
    transcript: list[TranscriptSegment],
    content_start: float,
    content_end: float,
    target_words_per_second: float = 3.0,
) -> float:
    words = sum(
        len(segment.text.split())
        for segment in transcript
        if segment.end > content_start and segment.start < content_end
    )
    duration = max(0.1, content_end - content_start)
    words_per_second = words / duration
    if words_per_second <= 3.2:
        return 1.0
    return round(max(0.85, min(1.0, target_words_per_second / words_per_second)), 3)


def run() -> None:
    processor = AnalysisJobProcessor()
    render_processor = RenderJobProcessor()
    try:
        processor.store.client.ping()
        render_processor.store.client.ping()
        recovered_analysis = processor.store.recover_interrupted()
        recovered_renders = render_processor.store.recover_interrupted()
        if recovered_analysis:
            logger.warning("Recovered %s interrupted analysis job(s)", recovered_analysis)
        if recovered_renders:
            logger.warning("Recovered %s interrupted render job(s)", recovered_renders)
        logger.info("Background worker is ready")
        while not shutdown.is_set():
            try:
                job_id = processor.store.wait(timeout=2)
                if job_id:
                    if shutdown.is_set():
                        processor.store.release(job_id)
                        break
                    _run_claimed_job(processor, job_id)
                if shutdown.is_set():
                    break
                render_job_id = render_processor.store.wait(timeout=1)
                if render_job_id:
                    if shutdown.is_set():
                        render_processor.store.release(render_job_id)
                        break
                    _run_claimed_job(render_processor, render_job_id)
            except RedisError:
                logger.exception("Redis unavailable; retrying in 2 seconds")
                shutdown.wait(2)
    except ForcedShutdown:
        logger.warning("Worker force-stopped; active work was returned to its queue")
    finally:
        logger.info("Closing Redis connections")
        for client in (processor.store.client, render_processor.store.client):
            try:
                client.close()
            except Exception:
                logger.exception("Could not close Redis connection")
        logger.info("Background worker stopped gracefully")


if __name__ == "__main__":
    install_signal_handlers()
    run()
