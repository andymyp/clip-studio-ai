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
    license: str = ""
    reusable: bool = False


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
    license: str = ""
    reusable: bool = False
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
    source_title: str = ""
    platform_profile: str = Field(
        default="smart",
        pattern="^(smart|youtube|tiktok|instagram)$",
    )
    content_style: str = Field(
        default="auto",
        pattern="^(auto|talking_head|gameplay|comedy|emotional|livestream|cinematic)$",
    )
    rights_confirmed: bool = False


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
    layouts: list[str] = Field(default_factory=list)
    zooms: list[float] = Field(default_factory=list)
    removed_seconds: float = 0
    hook: str = ""
    pattern_interrupts: list[float] = Field(default_factory=list)
    important_phrases: list[str] = Field(default_factory=list)
    playback_speed: float = 1.0
    audio_loudness_lufs: float | None = None
    platform_profile: str = "smart"
    content_style: str = "auto"


class QualityReport(BaseModel):
    passed: bool = False
    width: int = 0
    height: int = 0
    duration: float = 0
    has_audio: bool = False
    audio_video_drift: float = 0
    black_frame_ratio: float = 0
    frozen_frame_ratio: float = 0
    warnings: list[str] = Field(default_factory=list)


class RenderJobState(RenderRequest):
    status: str = "queued"
    progress: float = 0
    message: str = "Queued for rendering"
    output_path: str = ""
    media_url: str = ""
    subtitle_path: str = ""
    marketing: MarketingMetadata | None = None
    optimization: OptimizationPlan | None = None
    quality: QualityReport | None = None
    error: str | None = None
