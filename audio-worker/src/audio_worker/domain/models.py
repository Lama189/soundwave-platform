from dataclasses import dataclass
from pathlib import Path
from uuid import UUID


@dataclass(frozen=True)
class TrackUploadEvent:
    """Incoming event from RabbitMQ catalog-events (track.uploaded)."""
    track_id: UUID
    artist_id: UUID
    raw_key: str


@dataclass(frozen=True)
class AudioMetadata:
    """Metadata extracted from the raw audio via ffprobe."""
    duration_seconds: int
    format_name: str
    bit_rate: int | None = None
    sample_rate: int | None = None
    channels: int | None = None
    has_embedded_cover: bool = False


@dataclass(frozen=True)
class TranscodedMedia:
    """Artifacts produced by local ffmpeg processing."""
    audio_file_path: Path
    preview_file_path: Path
    cover_file_path: Path | None
    duration_seconds: int


@dataclass(frozen=True)
class ProcessingResult:
    """Final outcome after successful processing and upload to MinIO."""
    track_id: UUID
    duration_seconds: int
    file_key: str
    preview_key: str
    cover_key: str | None = None


@dataclass(frozen=True)
class ProcessingFailure:
    """Details when processing cannot be completed."""
    track_id: UUID
    error_message: str
