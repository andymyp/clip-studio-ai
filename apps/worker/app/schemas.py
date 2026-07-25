from pydantic import BaseModel, Field, HttpUrl


class TranscriptSegment(BaseModel):
    text: str
    start: float = Field(ge=0)
    end: float = Field(gt=0)


class RankedClip(BaseModel):
    start: float = Field(ge=0)
    end: float = Field(gt=0)
    score: float = Field(ge=0, le=100)
    reason: str


class AnalyzeVideoRequest(BaseModel):
    user_id: str
    external_id: str
    url: HttpUrl
    title: str
    platform: str
    thumbnail: str = ""
    youtube_username: str = ""


class GeneratedClip(RankedClip):
    id: str
    output_path: str
    media_url: str


class AnalysisJob(BaseModel):
    id: str
    user_id: str
    external_id: str = ""
    url: str
    title: str
    platform: str
    thumbnail: str = ""
    youtube_username: str = ""
    status: str
    progress: float
    message: str
    clips: list[GeneratedClip] = Field(default_factory=list)
    error: str | None = None


class RenderRequest(BaseModel):
    id: str
    user_id: str
    analysis_job_id: str
    clip_id: str
    clip_path: str
    start: float
    end: float
    watermark_text: str = ""
    source_url: str
    source_username: str = ""


class MarketingMetadata(BaseModel):
    title: str
    description: str
    hashtags: list[str] = Field(default_factory=list)
    hook: str = ""


class EditInterval(BaseModel):
    start: float = Field(ge=0)
    end: float = Field(gt=0)


class OptimizationPlan(BaseModel):
    intervals: list[EditInterval]
    face_centers: list[float] = Field(default_factory=list)
    removed_seconds: float = 0
    hook: str = ""
    pattern_interrupts: list[float] = Field(default_factory=list)
    important_phrases: list[str] = Field(default_factory=list)
    playback_speed: float = 1.0


class RenderJobState(RenderRequest):
    status: str = "queued"
    progress: float = 0
    message: str = "Queued for rendering"
    output_path: str = ""
    media_url: str = ""
    subtitle_path: str = ""
    marketing: MarketingMetadata | None = None
    optimization: OptimizationPlan | None = None
    error: str | None = None
