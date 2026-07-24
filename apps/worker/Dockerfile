FROM python:3.14-slim AS runtime

ENV PYTHONDONTWRITEBYTECODE=1 \
    PYTHONUNBUFFERED=1 \
    PIP_NO_CACHE_DIR=1 \
    PATH="/opt/venv/bin:$PATH"

RUN apt-get update \
    && apt-get install -y --no-install-recommends ffmpeg curl tini \
    && rm -rf /var/lib/apt/lists/*

RUN python -m venv /opt/venv
WORKDIR /app

COPY pyproject.toml ./
COPY app ./app
RUN pip install --upgrade pip && pip install .

RUN useradd --create-home --uid 10001 app \
    && mkdir -p /app/storage \
    && chown -R app:app /app

USER app
EXPOSE 3002
ENTRYPOINT ["tini", "--"]
CMD ["uvicorn", "app.main:app", "--host", "0.0.0.0", "--port", "3002"]
