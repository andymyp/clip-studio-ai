from importlib import import_module
from pathlib import Path
from statistics import median

from app.schemas import EditInterval


class SubjectReframeService:
    """Plans a full-screen crop from active faces, objects, and action."""

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
                [1.04] * len(intervals),
            )
        capture = cv2.VideoCapture(str(clip))
        detector = cv2.CascadeClassifier(
            cv2.data.haarcascades + "haarcascade_frontalface_default.xml"
        )
        centers: list[float] = []
        layouts: list[str] = []
        zooms: list[float] = []
        try:
            for interval in intervals:
                face_samples: list[tuple[float, float]] = []
                action_samples: list[tuple[float, float]] = []
                object_samples: list[tuple[float, float]] = []
                duration = interval.end - interval.start
                previous_gray = None
                previous_mouth = None
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
                        gray, scaleFactor=1.1, minNeighbors=6, minSize=(48, 48)
                    )
                    for x, y, width, height in faces:
                        mouth = gray[
                            y + height // 2 : y + height,
                            x : x + width,
                        ]
                        mouth = cv2.resize(mouth, (32, 16))
                        mouth_activity = (
                            float(cv2.absdiff(mouth, previous_mouth).mean())
                            if previous_mouth is not None
                            else 0
                        )
                        area = width * height / max(1, frame.shape[0] * frame.shape[1])
                        score = area * (1 + min(2.5, mouth_activity / 12))
                        face_samples.append(
                            ((x + width / 2) / frame.shape[1], score)
                        )
                        if not face_samples or score >= max(
                            value[1] for value in face_samples
                        ):
                            previous_mouth = mouth

                    if previous_gray is not None:
                        difference = cv2.absdiff(gray, previous_gray)
                        _threshold, moving = cv2.threshold(
                            difference, 24, 255, cv2.THRESH_BINARY
                        )
                        moments = cv2.moments(moving)
                        if moments["m00"] > frame.shape[0] * frame.shape[1] * 2:
                            action_samples.append(
                                (
                                    moments["m10"]
                                    / moments["m00"]
                                    / frame.shape[1],
                                    moments["m00"],
                                )
                            )
                    edges = cv2.Canny(gray, 80, 180)
                    contours, _hierarchy = cv2.findContours(
                        edges, cv2.RETR_EXTERNAL, cv2.CHAIN_APPROX_SIMPLE
                    )
                    for contour in contours:
                        x, _y, width, height = cv2.boundingRect(contour)
                        ratio = width * height / max(
                            1, frame.shape[0] * frame.shape[1]
                        )
                        if 0.01 <= ratio <= 0.55:
                            object_samples.append(
                                ((x + width / 2) / frame.shape[1], ratio)
                            )
                    previous_gray = gray

                if face_samples:
                    strongest = sorted(
                        face_samples, key=lambda value: value[1], reverse=True
                    )[:5]
                    center = median(value[0] for value in strongest)
                    zoom = 1.12
                elif action_samples:
                    strongest = sorted(
                        action_samples, key=lambda value: value[1], reverse=True
                    )[:3]
                    center = median(value[0] for value in strongest)
                    zoom = 1.08
                elif object_samples:
                    strongest = sorted(
                        object_samples, key=lambda value: value[1], reverse=True
                    )[:3]
                    center = median(value[0] for value in strongest)
                    zoom = 1.05
                else:
                    center, zoom = 0.5, 1.02
                centers.append(max(0.15, min(0.85, center)))
                layouts.append("crop")
                zooms.append(zoom)
        finally:
            capture.release()
        return _smooth_centers(centers), layouts, _smooth_zooms(zooms)


def _smooth_centers(centers: list[float]) -> list[float]:
    if not centers:
        return []
    smoothed = [centers[0]]
    for center in centers[1:]:
        previous = smoothed[-1]
        delta = center - previous
        # Hold the current shot unless the speaker has moved materially.
        if abs(delta) < 0.075:
            smoothed.append(previous)
            continue
        # Cap movement per semantic shot to avoid a floating automated camera.
        smoothed.append(max(0.15, min(0.85, previous + max(-0.065, min(0.065, delta)))))
    return smoothed


def _smooth_zooms(zooms: list[float]) -> list[float]:
    if not zooms:
        return []
    smoothed = [zooms[0]]
    for zoom in zooms[1:]:
        previous = smoothed[-1]
        smoothed.append(previous + max(-0.035, min(0.035, zoom - previous)))
    return smoothed
