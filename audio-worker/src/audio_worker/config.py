from pathlib import Path
from pydantic_settings import BaseSettings, SettingsConfigDict


class Settings(BaseSettings):
    rabbitmq_url: str = "amqp://guest:guest@localhost:5672"
    rabbitmq_exchange: str = "catalog-events"
    rabbitmq_queue: str = "audio-worker-track-uploaded"
    rabbitmq_routing_key: str = "track.uploaded"

    minio_endpoint: str = "localhost:9000"
    minio_access_key: str = "minioadmin"
    minio_secret_key: str = "minioadmin"
    minio_secure: bool = False

    minio_raw_bucket: str = "soundwave-raw"
    minio_audio_bucket: str = "soundwave-audio"
    minio_covers_bucket: str = "soundwave-covers"

    temp_dir: Path = Path("/tmp/soundwave-audio-worker")
    log_level: str = "INFO"

    audio_bitrate_kbps: int = 320
    audio_sample_rate_hz: int = 44100
    preview_duration_seconds: int = 30

    model_config = SettingsConfigDict(
        env_file=".env",
        env_file_encoding="utf-8",
        extra="ignore",
    )


settings = Settings()
