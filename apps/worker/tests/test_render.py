from pathlib import Path
from unittest.mock import patch

from app.schemas import EditInterval
from app.services.render import RenderService


def test_render_builds_vertical_ffmpeg_pipeline(tmp_path: Path) -> None:
    clip = tmp_path / "clip.mp4"
    subtitle = tmp_path / "subtitle.ass"
    output = tmp_path / "final.mp4"
    clip.touch()
    subtitle.touch()
    progress: list[float] = []

    def run(command: list[str], **_kwargs: object) -> object:
        output.touch()
        graph = command[command.index("-filter_complex") + 1]
        assert graph.count("scale=-2:1920:flags=lanczos") == 2
        assert "scale=-2:2016" not in graph
        assert "t/7.000" in graph
        assert "crop=1080:1920" in graph
        assert graph.count("setsar=1") == 2
        assert graph.count("fps=30000/1001") == 2
        assert graph.count("format=yuv420p") == 2
        assert graph.count("aresample=async=1:first_pts=0") == 2
        assert "volume=0:enable='between(t,0,2.000)'" in graph
        assert "volume=0:enable='gte(t,12.800)'" in graph
        assert "drawtext=textfile=" in graph
        assert "watermark.txt" in graph
        assert "hook.txt" in graph
        assert "y=(h-text_h)/2" in graph
        assert "borderw=7" in graph
        assert "text_align=C" in graph
        assert "line_spacing=4" in graph
        return type("Result", (), {"returncode": 0, "stderr": ""})()

    with patch("app.services.render.subprocess.run", side_effect=run):
        result = RenderService().render(
            clip, subtitle, output, "ClipStudio", "https://youtube.com/watch?v=x",
            lambda value, _message: progress.append(value),
            [EditInterval(start=0, end=7), EditInterval(start=7, end=14)],
            [0.25, 0.75],
            "This changes everything",
            "@ClipVerse.Studio",
        )

    assert result == output
    assert progress == [55, 90]
    assert (tmp_path / "watermark.txt").read_text() == "ClipStudio"
    assert (tmp_path / "hook.txt").read_text() == "This changes\neverything"
    assert (tmp_path / "source.txt").read_text() == "Source: YT @ClipVerse.Studio"
    assert "\n" not in (tmp_path / "source.txt").read_text()


def test_render_slows_video_audio_and_silence_timing_together(tmp_path: Path) -> None:
    clip, subtitle, output = tmp_path / "clip.mp4", tmp_path / "subtitle.ass", tmp_path / "out.mp4"
    clip.touch()
    subtitle.touch()

    def run(command: list[str], **_kwargs: object) -> object:
        output.touch()
        graph = command[command.index("-filter_complex") + 1]
        assert "setpts=(PTS-STARTPTS)/0.850" in graph
        assert "atempo=0.850" in graph
        assert "between(t,0,2.353)" in graph
        assert "gte(t,12.706)" in graph
        return type("Result", (), {"returncode": 0, "stderr": ""})()

    with patch("app.services.render.subprocess.run", side_effect=run):
        RenderService().render(
            clip,
            subtitle,
            output,
            "",
            "",
            lambda *_: None,
            [EditInterval(start=0, end=12)],
            [0.5],
            "Readable hook",
            "@creator",
            2,
            1.2,
            0.85,
        )
