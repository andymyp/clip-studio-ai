import pytest

from app.services.reframe import (
    _is_valid_face,
    _primary_speaker_center,
    _smooth_centers,
)


def test_smooth_centers_ignores_small_face_detector_jitter() -> None:
    assert _smooth_centers([0.5, 0.55, 0.57]) == [0.5, 0.5, 0.5]


def test_smooth_centers_requires_confirmation_for_nearby_reframe() -> None:
    result = _smooth_centers([0.5, 0.65, 0.66])

    assert result[:2] == [0.5, 0.5]
    assert result[2] == pytest.approx(0.655)


def test_smooth_centers_switches_immediately_to_distant_speaker() -> None:
    assert _smooth_centers([0.2, 0.8]) == [0.2, 0.8]


def test_primary_speaker_prefers_consistently_detected_face() -> None:
    samples = [(0.25, 1.0), (0.27, 1.0), (0.26, 0.8), (0.75, 5.0)]

    assert _primary_speaker_center(samples) == pytest.approx(0.26)


def test_face_validation_rejects_low_hand_sized_false_positive() -> None:
    assert not _is_valid_face(700, 800, 90, 90, 1920, 1080)


def test_face_validation_accepts_head_in_upper_frame() -> None:
    assert _is_valid_face(700, 180, 180, 200, 1920, 1080)
