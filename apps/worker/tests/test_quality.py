import json
from pathlib import Path
from unittest.mock import patch

from app.services.quality import RenderQualityService


def test_quality_control_accepts_valid_vertical_synchronized_render(
    tmp_path: Path,
) -> None:
    probe = type(
        "Result",
        (),
        {
            "returncode": 0,
            "stderr": "",
            "stdout": json.dumps(
                {
                    "streams": [
                        {
                            "codec_type": "video",
                            "width": 1080,
                            "height": 1920,
                            "duration": "30.00",
                        },
                        {"codec_type": "audio", "duration": "30.04"},
                    ],
                    "format": {"duration": "30.04"},
                }
            ),
        },
    )()
    scan = type("Result", (), {"returncode": 0, "stderr": "", "stdout": ""})()

    with patch("app.services.quality.subprocess.run", side_effect=[probe, scan]):
        report = RenderQualityService().inspect(tmp_path / "final.mp4", 30)

    assert report.passed
    assert report.width == 1080
    assert report.height == 1920
    assert report.audio_video_drift == 0.04
