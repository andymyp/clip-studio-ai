from importlib import import_module
from pathlib import Path
from statistics import median

from app.schemas import EditInterval


class SubjectReframeService:
    """Plans a stable, head-safe crop locked to the primary speaker."""

    def centers(self, clip: Path, intervals: list[EditInterval]) -> list[float]:
        return self.plan(clip, intervals)[0]

    def plan(
        self, clip: Path, intervals: list[EditInterval]
    ) -> tuple[list[float], list[str], list[float]]:
        try:
            cv2 = import_module("cv2")
        except ModuleNotFoundError:
            return (
                [0.5] * len(intervals),
                ["crop"] * len(intervals),
                [1.0] * len(intervals),
            )
        capture = cv2.VideoCapture(str(clip))
        detector = cv2.CascadeClassifier(
            cv2.data.haarcascades + "haarcascade_frontalface_default.xml"
        )
        eye_detector = cv2.CascadeClassifier(
            cv2.data.haarcascades + "haarcascade_eye_tree_eyeglasses.xml"
        )
        centers: list[float] = []
        layouts: list[str] = []
        zooms: list[float] = []
        try:
            for interval in intervals:
                face_samples: list[tuple[float, float]] = []
                duration = interval.end - interval.start
                previous_mouths: dict[int, object] = {}
                for fraction in (0.12, 0.31, 0.5, 0.69, 0.88):
                    capture.set(
                        cv2.CAP_PROP_POS_MSEC,
                        (interval.start + duration * fraction) * 1000,
                    )
                    ok, frame = capture.read()
                    if not ok:
                        continue
                    gray = cv2.cvtColor(frame, cv2.COLOR_BGR2GRAY)
                    faces = detector.detectMultiScale(
                        gray, scaleFactor=1.08, minNeighbors=8, minSize=(48, 48)
                    )
                    for x, y, width, height in faces:
                        if not _is_valid_face(
                            x,
                            y,
                            width,
                            height,
                            frame.shape[1],
                            frame.shape[0],
                        ):
                            continue
                        upper_face = gray[
                            y : y + max(1, round(height * 0.65)),
                            x : x + width,
                        ]
                        eyes = eye_detector.detectMultiScale(
                            upper_face,
                            scaleFactor=1.08,
                            minNeighbors=5,
                            minSize=(
                                max(8, width // 10),
                                max(8, height // 10),
                            ),
                        )
                        # Eye validation prevents hands and textured objects from
                        # becoming a confident camera target.
                        if len(eyes) == 0:
                            continue
                        normalized_center = (x + width / 2) / frame.shape[1]
                        track_key = round(normalized_center * 10)
                        mouth = gray[
                            y + height // 2 : y + height,
                            x : x + width,
                        ]
                        mouth = cv2.resize(mouth, (32, 16))
                        previous_mouth = previous_mouths.get(track_key)
                        mouth_activity = (
                            float(cv2.absdiff(mouth, previous_mouth).mean())
                            if previous_mouth is not None
                            else 0
                        )
                        area = width * height / max(1, frame.shape[0] * frame.shape[1])
                        score = area * (1 + min(2.5, mouth_activity / 12))
                        face_samples.append((normalized_center, score))
                        previous_mouths[track_key] = mouth

                if face_samples:
                    center = _primary_speaker_center(face_samples)
                else:
                    # A neutral crop is safer than following an unverified hand or
                    # object. Later smoothing keeps an established speaker locked.
                    center = 0.5
                centers.append(max(0.08, min(0.92, center)))
                layouts.append("crop")
                # Preserve the complete source height so forehead/hair cannot be
                # removed by a vertically centered zoom crop.
                zooms.append(1.0)
        finally:
            capture.release()
        return _smooth_centers(centers), layouts, _smooth_zooms(zooms)


def _smooth_centers(centers: list[float]) -> list[float]:
    if not centers:
        return []
    locked = centers[0]
    smoothed = [locked]
    pending: float | None = None
    confirmations = 0
    for center in centers[1:]:
        delta = center - locked
        if abs(delta) < 0.10:
            pending = None
            confirmations = 0
            smoothed.append(locked)
            continue
        # A large displacement is a clear cut to another speaker. Smaller
        # changes require confirmation across two beats to reject detector jitter.
        if abs(delta) >= 0.28:
            locked = center
            pending = None
            confirmations = 0
        elif pending is not None and abs(center - pending) < 0.08:
            confirmations += 1
            pending = median((pending, center))
            if confirmations >= 2:
                locked = pending
                pending = None
                confirmations = 0
        else:
            pending = center
            confirmations = 1
        smoothed.append(locked)
    return smoothed


def _smooth_zooms(zooms: list[float]) -> list[float]:
    if not zooms:
        return []
    smoothed = [zooms[0]]
    for zoom in zooms[1:]:
        previous = smoothed[-1]
        smoothed.append(previous + max(-0.035, min(0.035, zoom - previous)))
    return smoothed


def _primary_speaker_center(
    samples: list[tuple[float, float]],
    track_distance: float = 0.12,
) -> float:
    tracks: list[list[tuple[float, float]]] = []
    for sample in sorted(samples, key=lambda value: value[0]):
        for track in tracks:
            if abs(sample[0] - median(value[0] for value in track)) <= track_distance:
                track.append(sample)
                break
        else:
            tracks.append([sample])
    strongest = max(
        tracks,
        key=lambda track: (
            len(track),
            sum(value[1] for value in track),
        ),
    )
    return median(value[0] for value in strongest)


def _is_valid_face(
    x: int,
    y: int,
    width: int,
    height: int,
    frame_width: int,
    frame_height: int,
) -> bool:
    if frame_width <= 0 or frame_height <= 0 or width <= 0 or height <= 0:
        return False
    aspect_ratio = width / height
    center_y = (y + height / 2) / frame_height
    relative_height = height / frame_height
    return (
        0.72 <= aspect_ratio <= 1.38
        and center_y <= 0.72
        and relative_height >= 0.055
        and x >= 0
        and y >= 0
        and x + width <= frame_width
        and y + height <= frame_height
    )
