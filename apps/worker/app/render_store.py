from redis import Redis
from redis.exceptions import TimeoutError as RedisTimeoutError

from app.schemas import RenderJobState, RenderRequest


class RenderJobStore:
    def __init__(self, client: Redis, queue_name: str) -> None:
        self.client = client
        self.queue_name = queue_name
        self.processing_queue_name = f"{queue_name}:processing"

    def create(self, payload: RenderRequest) -> RenderJobState:
        job = RenderJobState(**payload.model_dump())
        self.save(job)
        self.client.lpush(self.queue_name, job.id)
        return job

    def get(self, job_id: str) -> RenderJobState | None:
        value = self.client.get(self._key(job_id))
        return RenderJobState.model_validate_json(value) if value else None

    def save(self, job: RenderJobState) -> None:
        self.client.set(self._key(job.id), job.model_dump_json(), ex=30 * 24 * 60 * 60)

    def wait(self, timeout: int = 1) -> str | None:
        try:
            value = self.client.brpoplpush(
                self.queue_name,
                self.processing_queue_name,
                timeout=timeout,
            )
        except RedisTimeoutError:
            return None
        if value is None:
            return None
        job_id = value
        return job_id.decode() if isinstance(job_id, bytes) else str(job_id)

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
        return recovered

    def retry(self, job_id: str) -> RenderJobState | None:
        job = self.get(job_id)
        if job is None:
            return None
        job.status = "queued"
        job.progress = 0
        job.message = "Queued for retry"
        job.error = None
        self.save(job)
        self.client.lpush(self.queue_name, job.id)
        return job

    @staticmethod
    def _key(job_id: str) -> str:
        return f"clipstudio:render:job:{job_id}"
