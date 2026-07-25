from importlib import import_module
from pathlib import Path

from app.schemas import EditInterval


class SubjectReframeService:
    """Samples each edit beat and centers the 9:16 crop on the dominant face."""

    def centers(self, clip: Path, intervals: list[EditInterval]) -> list[float]:
        try:
            cv2 = import_module("cv2")
        except ModuleNotFoundError:
            return [0.5] * len(intervals)
        capture = cv2.VideoCapture(str(clip))
        detector = cv2.CascadeClassifier(
            cv2.data.haarcascades + "haarcascade_frontalface_default.xml"
        )
        centers: list[float] = []
        try:
            for interval in intervals:
                capture.set(cv2.CAP_PROP_POS_MSEC, ((interval.start + interval.end) / 2) * 1000)
                ok, frame = capture.read()
                if not ok:
                    centers.append(0.5)
                    continue
                gray = cv2.cvtColor(frame, cv2.COLOR_BGR2GRAY)
                faces = detector.detectMultiScale(gray, scaleFactor=1.1, minNeighbors=5)
                if len(faces) == 0:
                    centers.append(0.5)
                    continue
                x, _y, width, _height = max(faces, key=lambda face: face[2] * face[3])
                centers.append(max(0.15, min(0.85, (x + width / 2) / frame.shape[1])))
        finally:
            capture.release()
        return _smooth_centers(centers)


def _smooth_centers(centers: list[float]) -> list[float]:
    if not centers:
        return []
    smoothed = [centers[0]]
    for center in centers[1:]:
        previous = smoothed[-1]
        delta = center - previous
        if abs(delta) < 0.035:
            smoothed.append(previous)
            continue
        smoothed.append(max(0.15, min(0.85, previous + max(-0.10, min(0.10, delta)))))
    return smoothed
