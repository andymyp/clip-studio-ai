from app.services.content_style import content_style_profile
from app.services.subtitle import _caption_limits


def test_gameplay_preserves_the_full_source_frame() -> None:
    profile = content_style_profile("gameplay")

    assert profile.framing == "fit"
    assert not profile.remove_fillers
    assert profile.minimum_speed == 1.0


def test_comedy_uses_tighter_pacing_than_emotional_content() -> None:
    comedy = content_style_profile("comedy")
    emotional = content_style_profile("emotional")

    assert comedy.silence_threshold < emotional.silence_threshold
    assert comedy.beat_maximum < emotional.beat_maximum
    assert comedy.remove_fillers
    assert not emotional.remove_fillers


def test_unknown_style_falls_back_to_auto() -> None:
    assert content_style_profile("unknown").key == "auto"


def test_caption_density_matches_content_tone() -> None:
    assert _caption_limits("comedy")[0] < _caption_limits("emotional")[0]
