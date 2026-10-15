import logging
import shutil
from pathlib import Path

from audio_worker.config import settings
from audio_worker.domain.models import ProcessingResult, TrackUploadEvent
from audio_worker.service.interfaces import AudioProber, MediaTranscoder, StorageClient

logger = logging.getLogger(__name__)


class AudioProcessor:

    def __init__(
        self, 
        storage: StorageClient, 
        prober: AudioProber, 
        transcoder: MediaTranscoder, 
        temp_dir: Path | None = None,
    ) -> None:
        self._storage = storage
        self._prober = prober
        self._transcoder = transcoder
        self._temp_dir = temp_dir or settings.temp_dir

    async def process(self, event: TrackUploadEvent) -> ProcessingResult:
        track_id = event.track_id
        work_dir = self._temp_dir / f"track_{track_id}"
        work_dir.mkdir(parents=True, exist_ok=True)

        logger.info("Starting processing track %s (raw_key: %s)", track_id, event.raw_key)

        try:
            file_ext = Path(event.raw_key).suffix or ".audio"
            raw_file_path = work_dir / f"raw_input{file_ext}"

            await self._storage.download_raw_audio(event.raw_key, raw_file_path)
            logger.debug("[%s] Successfully downloaded raw file to temp directory", track_id)

            metadata = await self._prober.probe(raw_file_path)
            logger.info(                                                                                                          
                "[%s] Metadata: format=%s, duration=%ds, embedded_cover=%s",                                                    
                track_id,                                                                                                         
                metadata.format_name,                                                                                             
                metadata.duration_seconds,                                                                                        
                metadata.has_embedded_cover,                                                                                      
            )  

            output_dir = work_dir / "output"
            transcoded = await self._transcoder.transcode(raw_file_path, output_dir, metadata)
            logger.debug("[%s] Transcoding completes successfully", track_id)

            file_key = await self._storage.upload_processed_audio(track_id, transcoded.audio_file_path)
            preview_key = await self._storage.upload_preview_audio(track_id, transcoded.preview_file_path)

            cover_key: str | None = None
            if transcoded.cover_file_path is not None:
                cover_key = await self._storage.upload_cover(track_id, transcoded.cover_file_path)
                logger.info("[%s] Extracted cover art uploaded to %s", track_id, cover_key)

            logger.info(
                "[%s] Processing successfully completed: file_key=%s, duration=%ds",
                track_id,
                file_key,
                metadata.duration_seconds
            )

            return ProcessingResult(
                track_id=track_id,
                duration_seconds=metadata.duration_seconds,
                file_key=file_key,
                preview_key=preview_key,
                cover_key=cover_key
            )

        finally:
            shutil.rmtree(work_dir, ignore_errors=True)
            logger.debug("[%s] Temporary directory cleared", track_id)