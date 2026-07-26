import re
import subprocess
from collections.abc import Callable
from pathlib import Path

from app.schemas import EditInterval
from app.services.audio import mastering_filter


class RenderError(RuntimeError):
    pass


ProgressCallback = Callable[[float, str], None]


class RenderService:
    def render(
        self,
        clip: Path,
        subtitle: Path,
        output: Path,
        watermark: str,
        source_url: str,
        progress: ProgressCallback,
        intervals: list[EditInterval] | None = None,
        face_centers: list[float] | None = None,
        hook: str = "",
        source_username: str = "",
        intro_silence: float = 2.0,
        outro_silence: float = 1.2,
        playback_speed: float = 1.0,
        audio_measurement: dict[str, float] | None = None,
        platform_profile: str = "youtube",
        layouts: list[str] | None = None,
        zooms: list[float] | None = None,
    ) -> Path:
        output.parent.mkdir(parents=True, exist_ok=True)
        progress(55, "Rendering retention-optimized vertical video")
        intervals = intervals or [EditInterval(start=0, end=86_400)]
        centers = face_centers or [0.5] * len(intervals)
        layouts = layouts or ["crop"] * len(intervals)
        zooms = zooms or [1.04] * len(intervals)
        graph: list[str] = []
        concat_inputs: list[str] = []
        for index, interval in enumerate(intervals):
            center = centers[index] if index < len(centers) else 0.5
            previous = centers[index - 1] if index > 0 and index - 1 < len(centers) else center
            duration = max(0.001, (interval.end - interval.start) / playback_speed)
            phase = f"min(1,max(0,t/{duration:.3f}))"
            easing = f"({phase})*({phase})*(3-2*({phase}))"
            pan = (
                f"{previous:.4f}+({center:.4f}-{previous:.4f})"
                f"*({easing})"
            )
            target_zoom = zooms[index] if index < len(zooms) else 1.04
            previous_zoom = (
                zooms[index - 1]
                if index > 0 and index - 1 < len(zooms)
                else 1.0
            )
            zoom = f"{previous_zoom:.4f}+({target_zoom:.4f}-{previous_zoom:.4f})*({easing})"
            scale_height = f"1920*({zoom})"
            opening_fade = ",fade=t=in:st=0:d=0.18" if index == 0 else ""
            if index < len(layouts) and layouts[index] == "fit":
                graph.extend(
                    [
                        (
                            f"[0:v]trim=start={interval.start:.3f}:end={interval.end:.3f},"
                            f"setpts=(PTS-STARTPTS)/{playback_speed:.3f},"
                            f"split=2[bgraw{index}][fgraw{index}]"
                        ),
                        (
                            f"[bgraw{index}]scale=1080:1920:"
                            "force_original_aspect_ratio=increase:flags=lanczos,"
                            f"crop=1080:1920,boxblur=30:10[bg{index}]"
                        ),
                        (
                            f"[fgraw{index}]scale=1080:1920:"
                            f"force_original_aspect_ratio=decrease:flags=lanczos[fg{index}]"
                        ),
                        (
                            f"[bg{index}][fg{index}]overlay=(W-w)/2:(H-h)/2,"
                            "setsar=1,fps=30000/1001,format=yuv420p"
                            f"{opening_fade}[v{index}]"
                        ),
                    ]
                )
            else:
                graph.append(
                    f"[0:v]trim=start={interval.start:.3f}:end={interval.end:.3f},"
                    f"setpts=(PTS-STARTPTS)/{playback_speed:.3f},"
                    f"scale=-2:'{scale_height}':flags=lanczos:eval=frame,"
                    f"crop=1080:1920:x='max(0,min(iw-1080,(iw-1080)*({pan})))':"
                    f"y=(ih-1920)/2,setsar=1,fps=30000/1001,format=yuv420p"
                    f"{opening_fade}"
                    f"[v{index}]"
                )
            graph.append(
                f"[0:a]atrim=start={interval.start:.3f}:end={interval.end:.3f},"
                f"asetpts=PTS-STARTPTS,atempo={playback_speed:.3f},"
                f"aresample=async=1:first_pts=0[a{index}]"
            )
            concat_inputs.append(f"[v{index}][a{index}]")
        graph.append(
            "".join(concat_inputs)
            + f"concat=n={len(intervals)}:v=1:a=1[vcat][acat]"
        )
        output_duration = sum(item.end - item.start for item in intervals) / playback_speed
        rendered_intro = intro_silence / playback_speed
        rendered_outro = outro_silence / playback_speed
        outro_start = max(rendered_intro, output_duration - rendered_outro)
        graph.append(
            f"[acat]{mastering_filter(audio_measurement or {})},"
            f"volume=0:enable='between(t,0,{rendered_intro:.3f})',"
            f"volume=0:enable='gte(t,{outro_start:.3f})'[aout]"
        )
        overlays = [f"ass='{_escape_filter_path(subtitle)}'"]
        if hook.strip():
            hook_path = _write_title_overlay(output.parent / "hook.txt", hook)
            hook_text = hook_path.read_text(encoding="utf-8")
            overlays.append(
                _drawtext_file(
                    hook_path,
                    "(h-text_h)/2",
                    size=_fit_title_font_size(hook_text),
                    enable=f"between(t,0,{rendered_intro:.3f})",
                    border_width=7,
                    line_spacing=-6,
                    text_align="C",
                )
            )
        if watermark.strip():
            overlays.append(
                _drawtext_file(
                    _write_overlay(output.parent / "watermark.txt", watermark, wrap=False),
                    65,
                    size=48,
                    boxed=True,
                    x="w-text_w-65",
                )
            )
        source = source_username.strip() or "YouTube"
        source_bottom = {
            "youtube": 95,
            "tiktok": 165,
            "instagram": 135,
        }.get(platform_profile, 95)
        overlays.append(
            _drawtext_file(
                _write_overlay(
                    output.parent / "source.txt",
                    f"Source: YT {source}",
                    wrap=False,
                ),
                f"h-text_h-{source_bottom}",
                size=40,
                boxed=True,
            )
        )
        graph.append(f"[vcat]{','.join(overlays)}[vout]")
        command = [
            "ffmpeg", "-y", "-i", str(clip), "-filter_complex", ";".join(graph),
            "-map", "[vout]", "-map", "[aout]", "-c:v", "libx264", "-preset", "slow",
            "-crf", "17", "-profile:v", "high", "-level:v", "4.2", "-pix_fmt", "yuv420p",
            "-c:a", "aac", "-b:a", "192k", "-ar", "48000", "-movflags", "+faststart",
            str(output),
        ]
        completed = subprocess.run(
            command, capture_output=True, text=True, timeout=900, check=False
        )
        if completed.returncode != 0 or not output.is_file():
            raise RenderError(completed.stderr.strip() or "FFmpeg render failed")
        progress(90, "Best-potential video render complete")
        return output


def _escape_filter_path(path: Path) -> str:
    return str(path.resolve()).replace("\\", "/").replace(":", r"\:").replace("'", r"\'")


def _write_overlay(path: Path, text: str, wrap: bool = True) -> Path:
    value = _wrap_overlay(text.strip()) if wrap else text.strip().replace("\r", " ").replace("\n", " ")
    path.write_text(value, encoding="utf-8")
    return path


def _write_title_overlay(path: Path, text: str) -> Path:
    clean = re.sub(r"[*_`#]+", "", text)
    clean = " ".join(clean.replace("\r", " ").replace("\n", " ").split())
    clean = clean.rstrip(" .,:;!-")
    path.write_text(_wrap_title(clean), encoding="utf-8")
    return path


def _drawtext_file(
    path: Path,
    y: int | str,
    size: int = 34,
    enable: str = "",
    boxed: bool = False,
    x: str = "(w-text_w)/2",
    border_width: int = 3,
    line_spacing: int = 10,
    text_align: str = "",
) -> str:
    suffix = f":enable='{enable}'" if enable else ""
    box = ":box=1:boxcolor=black@0.48:boxborderw=16" if boxed else ""
    alignment = f":text_align={text_align}" if text_align else ""
    return (
        f"drawtext=textfile='{_escape_filter_path(path)}':reload=0:"
        f"font='Arial':fontcolor=white:fontsize={size}:"
        f"borderw={border_width}:bordercolor=black:line_spacing={line_spacing}:"
        f"x={x}:y={y}{alignment}{box}{suffix}"
    )


def _fit_title_font_size(wrapped_text: str, maximum: int = 128, minimum: int = 72) -> int:
    longest = max((len(line) for line in wrapped_text.splitlines()), default=1)
    return max(minimum, min(maximum, round(2_000 / max(longest, 1))))


def _wrap_title(text: str, maximum_lines: int = 2) -> str:
    words = text.replace("\r", " ").replace("\n", " ").split()
    if not words:
        return ""

    line_count = min(maximum_lines, max(1, round(len(" ".join(words)) / 15)))
    lines: list[str] = []
    remaining = words
    for index in range(line_count):
        lines_left = line_count - index
        if lines_left == 1:
            lines.append(" ".join(remaining))
            break
        target = max(1, round(len(" ".join(remaining)) / lines_left))
        take = 1
        while take < len(remaining) - (lines_left - 1):
            current = len(" ".join(remaining[:take]))
            candidate = len(" ".join(remaining[: take + 1]))
            if abs(candidate - target) > abs(current - target):
                break
            take += 1
        lines.append(" ".join(remaining[:take]))
        remaining = remaining[take:]
    return "\n".join(lines)


def _wrap_overlay(text: str, width: int = 26, maximum_lines: int = 2) -> str:
    words = text.split()
    lines: list[str] = []
    current: list[str] = []
    for word in words:
        candidate = " ".join([*current, word])
        if current and len(candidate) > width and len(lines) < maximum_lines - 1:
            lines.append(" ".join(current))
            current = []
        current.append(word)
    if current:
        lines.append(" ".join(current))
    return "\n".join(lines[:maximum_lines])
