from uuid import uuid4

from redis import Redis
from redis.exceptions import TimeoutError as RedisTimeoutError

from app.schemas import AnalysisJob, AnalyzeVideoRequest


class AnalysisJobStore:
    def __init__(self, client: Redis, queue_name: str) -> None:
        self.client = client
        self.queue_name = queue_name
        self.processing_queue_name = f"{queue_name}:processing"

    def create(self, payload: AnalyzeVideoRequest) -> AnalysisJob:
        cached = self.get_review(payload.user_id, payload.platform, payload.external_id)
        if cached is not None and cached.status != "failed":
            return cached
        job = AnalysisJob(
            id=str(uuid4()),
            user_id=payload.user_id,
            external_id=payload.external_id,
            url=str(payload.url),
            title=payload.title,
            platform=payload.platform,
            thumbnail=payload.thumbnail,
            youtube_username=payload.youtube_username,
            license=payload.license,
            reusable=payload.reusable,
            status="queued",
            progress=0,
            message="Queued for subtitle extraction",
        )
        self.save(job)
        self.client.set(
            self._review_key(payload.user_id, payload.platform, payload.external_id),
            job.id,
            ex=30 * 24 * 60 * 60,
        )
        self.client.lpush(self.queue_name, job.id)
        return job

    def get(self, job_id: str) -> AnalysisJob | None:
        value = self.client.get(self._key(job_id))
        if not value:
            return None
        return AnalysisJob.model_validate_json(value)

    def save(self, job: AnalysisJob) -> None:
        ttl = 30 * 24 * 60 * 60 if job.status == "completed" else 7 * 24 * 60 * 60
        self.client.set(self._key(job.id), job.model_dump_json(), ex=ttl)
        if job.status == "completed":
            self.client.set(
                self._review_key(job.user_id, job.platform, job.external_id),
                job.id,
                ex=ttl,
            )

    def get_review(self, user_id: str, platform: str, external_id: str) -> AnalysisJob | None:
        job_id = self.client.get(self._review_key(user_id, platform, external_id))
        if not job_id:
            return None
        value = job_id.decode() if isinstance(job_id, bytes) else str(job_id)
        job = self.get(value)
        if job is None:
            self.client.delete(self._review_key(user_id, platform, external_id))
        return job

    def invalidate_review(self, job: AnalysisJob) -> None:
        self.client.delete(self._review_key(job.user_id, job.platform, job.external_id))

    def wait(self, timeout: int = 5) -> str | None:
        try:
            item = self.client.brpoplpush(
                self.queue_name,
                self.processing_queue_name,
                timeout=timeout,
            )
        except RedisTimeoutError:
            # redis-py can reach its socket-read boundary just before Redis
            # returns nil for an idle blocking pop. Treat that as an empty
            # queue, not as a crashed consumer.
            return None
        if item is None:
            return None
        return item.decode() if isinstance(item, bytes) else str(item)

    def acknowledge(self, job_id: str) -> None:
        self.client.lrem(self.processing_queue_name, 1, job_id)

    def release(self, job_id: str) -> None:
        # Only put the job back when it is still owned by this worker.  This
        # keeps forced shutdown idempotent if a signal races with acknowledge().
        self.client.eval(
            """
            if redis.call('LREM', KEYS[1], 1, ARGV[1]) == 1 then
                return redis.call('RPUSH', KEYS[2], ARGV[1])
            end
            return 0
            """,
            2,
            self.processing_queue_name,
            self.queue_name,
            job_id,
        )

    def recover_interrupted(self) -> int:
        recovered = 0
        while self.client.rpoplpush(self.processing_queue_name, self.queue_name) is not None:
            recovered += 1
        for key in self.client.scan_iter(match="clipstudio:analysis:job:*"):
            value = self.client.get(key)
            if not value:
                continue
            job = AnalysisJob.model_validate_json(value)
            if job.status not in {"queued", "processing"}:
                continue
            queued = self.client.lpos(self.queue_name, job.id) is not None
            processing = self.client.lpos(self.processing_queue_name, job.id) is not None
            if not queued and not processing:
                job.status = "queued"
                job.message = "Recovered after worker interruption"
                self.save(job)
                self.client.lpush(self.queue_name, job.id)
                recovered += 1
        return recovered

    @staticmethod
    def _key(job_id: str) -> str:
        return f"clipstudio:analysis:job:{job_id}"

    @staticmethod
    def _review_key(user_id: str, platform: str, external_id: str) -> str:
        # Version the ranking cache so algorithm changes do not serve stale scores.
        return f"clipstudio:analysis:review:v5:{user_id}:{platform}:{external_id}"
