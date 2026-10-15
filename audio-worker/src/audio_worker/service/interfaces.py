from pathlib import Path
from typing import Protocol
from uuid import UUID

from audio_worker.domain.models import (
    AudioMetadata,
    ProcessingFailure,
    ProcessingResult,
    TranscodedMedia,
)


class StorageClient(Protocol):

    async def download_raw_audio(self, raw_key: str, dest_path: Path) -> Path:
        """Download raw audio file by its object key."""
        ...

    async def upload_processed_audio(self, track_id: UUID, file_path: Path) -> str:
        """Upload the primary processed MP3 track and return file_key."""
        ...

    async def upload_preview_audio(self, track_id: UUID, file_path: Path) -> str:
        """Upload the 30-second preview audio and return preview_key."""
        ...

    async def upload_cover(self, track_id: UUID, file_path: Path) -> str:
        """Upload the cover image and return cover_key."""
        ...


class AudioProber(Protocol):

    async def probe(self, file_path: Path) -> AudioMetadata:
        """Extract technical metadata from the audio file."""
        ...


class MediaTranscoder(Protocol):

    async def transcode(
        self,
        input_path: Path,
        output_dir: Path,
        metadata: AudioMetadata,
    ) -> TranscodedMedia:
        """Transcode audio and generate required media artifacts."""
        ...


class EventPublisher(Protocol):

    async def publish_processed(self, result: ProcessingResult) -> None:
        """Publish a successful processing event (track.processed)."""
        ...

    async def publish_failed(self, failure: ProcessingFailure) -> None:
        """Publish a processing failure event (track.processing_failed)."""
        ...