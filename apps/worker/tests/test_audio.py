from app.services.audio import mastering_filter


def test_two_pass_mastering_filter_uses_measured_loudness_and_voice_processing() -> None:
    value = mastering_filter(
        {
            "input_i": -23.1,
            "input_lra": 4.2,
            "input_tp": -3.0,
            "input_thresh": -33.0,
            "target_offset": 0.2,
        }
    )

    assert "highpass=f=80" in value
    assert "afftdn=" in value
    assert "acompressor=" in value
    assert "measured_I=-23.10" in value
    assert "alimiter=limit=0.841" in value
