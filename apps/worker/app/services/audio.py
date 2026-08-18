import json
import re
import subprocess
from pathlib import Path


class AudioMasteringService:
    """Measures speech loudness for deterministic EBU R128 normalization."""

    def measure(self, clip: Path) -> dict[str, float]:
        command = [
            "ffmpeg",
            "-hide_banner",
            "-i",
            str(clip),
            "-af",
            (
                "highpass=f=80,afftdn=nf=-25,"
                "acompressor=threshold=-20dB:ratio=3:attack=15:release=180,"
                "loudnorm=I=-16:LRA=7:TP=-1.5:print_format=json"
            ),
            "-f",
            "null",
            "-",
        ]
        completed = subprocess.run(
            command, capture_output=True, text=True, timeout=180, check=False
        )
        matches = re.findall(r"\{[\s\S]*?\}", completed.stderr)
        if completed.returncode != 0 or not matches:
            return {}
        try:
            payload = json.loads(matches[-1])
            return {
                "input_i": float(payload["input_i"]),
                "input_lra": float(payload["input_lra"]),
                "input_tp": float(payload["input_tp"]),
                "input_thresh": float(payload["input_thresh"]),
                "target_offset": float(payload["target_offset"]),
            }
        except (KeyError, TypeError, ValueError, json.JSONDecodeError):
            return {}


def mastering_filter(measurement: dict[str, float]) -> str:
    base = (
        "highpass=f=80,afftdn=nf=-25,"
        "acompressor=threshold=-20dB:ratio=3:attack=15:release=180"
    )
    if not measurement:
        return f"{base},loudnorm=I=-16:LRA=7:TP=-1.5,alimiter=limit=0.841"
    return (
        f"{base},loudnorm=I=-16:LRA=7:TP=-1.5:"
        f"measured_I={measurement['input_i']:.2f}:"
        f"measured_LRA={measurement['input_lra']:.2f}:"
        f"measured_TP={measurement['input_tp']:.2f}:"
        f"measured_thresh={measurement['input_thresh']:.2f}:"
        f"offset={measurement['target_offset']:.2f}:linear=true,"
        "alimiter=limit=0.841"
    )
