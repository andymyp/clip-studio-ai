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
    status: str
    progress: float
    message: str
    clips: list[GeneratedClip] = Field(default_factory=list)
    error: str | None = None
