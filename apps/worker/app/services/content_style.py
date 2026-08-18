from dataclasses import dataclass


@dataclass(frozen=True)
class ContentStyleProfile:
    key: str
    silence_threshold: float
    breath_padding: float
    remove_fillers: bool
    beat_minimum: float
    beat_maximum: float
    framing: str
    minimum_speed: float


PROFILES = {
    "auto": ContentStyleProfile("auto", 0.65, 0.12, True, 4.0, 9.0, "speaker", 0.85),
    "talking_head": ContentStyleProfile("talking_head", 0.65, 0.12, True, 4.0, 9.0, "speaker", 0.85),
    "gameplay": ContentStyleProfile("gameplay", 1.25, 0.18, False, 7.0, 14.0, "fit", 1.0),
    "comedy": ContentStyleProfile("comedy", 0.45, 0.08, True, 2.5, 6.0, "speaker", 0.95),
    "emotional": ContentStyleProfile("emotional", 1.8, 0.28, False, 7.0, 14.0, "speaker", 0.85),
    "livestream": ContentStyleProfile("livestream", 0.5, 0.08, True, 3.0, 7.0, "speaker", 0.9),
    "cinematic": ContentStyleProfile("cinematic", 2.0, 0.3, False, 8.0, 16.0, "fit", 0.9),
}


def content_style_profile(value: str) -> ContentStyleProfile:
    return PROFILES.get(value, PROFILES["auto"])
