FROM python:3.13-slim

# Install system dependencies (ffmpeg and ffprobe)
RUN apt-get update && apt-get install -y --no-install-recommends \
    ffmpeg \
    ca-certificates \
    curl \
    && rm -rf /var/lib/apt/lists/*

# Install uv from official image
COPY --from=ghcr.io/astral-sh/uv:latest /uv /uvx /bin/

WORKDIR /app

ENV UV_COMPILE_BYTECODE=1 \
    PYTHONUNBUFFERED=1

# Copy dependency files first for layer caching
COPY pyproject.toml uv.lock .python-version ./

# Install dependencies into virtual environment
RUN uv sync --frozen --no-dev --no-install-project

# Copy source code
COPY README.md ./
COPY src/ ./src/

# Install the project itself
RUN uv sync --frozen --no-dev

# Put virtualenv binaries in PATH
ENV PATH="/app/.venv/bin:$PATH"

CMD ["python", "-m", "audio_worker"]
