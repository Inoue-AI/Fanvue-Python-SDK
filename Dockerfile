FROM python:3.12-slim

ENV PYTHONDONTWRITEBYTECODE=1 \
    PYTHONUNBUFFERED=1

WORKDIR /app

COPY pyproject.toml README.md LICENSE ./
COPY fanvue_sdk ./fanvue_sdk
COPY scripts ./scripts
COPY tests ./tests

RUN pip install --no-cache-dir -e .[dev]

CMD ["pytest", "-q"]
