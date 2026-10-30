import asyncio
import logging
from pathlib import Path
from uuid import UUID

from minio import Minio
from minio.error import S3Error

from audio_worker.config import settings
from audio_worker.domain.exceptions import StorageDownloadError, StorageUploadError

logger = logging.getLogger(__name__)


class MinioStorageClient:
    def __init__(
        self,
        endpoint: str | None = None,
        access_key: str | None = None,
        secret_key: str | None = None,
        secure: bool | None = None,
        raw_bucket: str | None = None,
        audio_bucket: str | None = None,
        covers_bucket: str | None = None,
    ) -> None:
        self._raw_bucket = raw_bucket or settings.minio_raw_bucket
        self._audio_bucket = audio_bucket or settings.minio_audio_bucket
        self._covers_bucket = covers_bucket or settings.minio_covers_bucket

        self._client = Minio(
            endpoint=endpoint or settings.minio_endpoint,
            access_key=access_key or settings.minio_access_key,
            secret_key=secret_key or settings.minio_secret_key,
            secure=secure if secure is not None else settings.minio_secure,
        )

    async def ensure_buckets(self) -> None:
        buckets = [self._raw_bucket, self._audio_bucket, self._covers_bucket]
        for bucket in buckets:
            try:
                exists = await asyncio.to_thread(self._client.bucket_exists, bucket)
                if not exists:
                    await asyncio.to_thread(self._client.make_bucket, bucket)
                    logger.info("MinIO bucket created: %s", bucket)
            except Exception as exc:  # noqa: BLE001
                logger.warning("Failed to check/create bucket %s: %s", bucket, exc)

    async def download_raw_audio(self, raw_key: str, dest_path: Path) -> Path:
        dest_path.parent.mkdir(parents=True, exist_ok=True)
        try:
            logger.debug("Downloading %s from bucket %s to %s", raw_key, self._raw_bucket, dest_path)
            await asyncio.to_thread(
                self._client.fget_object,
                bucket_name=self._raw_bucket,
                object_name=raw_key,
                file_path=str(dest_path),
            )
            return dest_path
        except S3Error as exc:
            raise StorageDownloadError(f"S3 error downloading {raw_key}: {exc.message}") from exc
        except Exception as exc:
            raise StorageDownloadError(f"Failed to download {raw_key} from MinIO: {exc}") from exc

    async def upload_processed_audio(self, track_id: UUID, file_path: Path) -> str:
        key = f"tracks/{track_id}.mp3"
        await self._upload_file(
            bucket=self._audio_bucket,
            object_name=key,
            file_path=file_path,
            content_type="audio/mpeg",
        )
        return key

    async def upload_preview_audio(self, track_id: UUID, file_path: Path) -> str:
        key = f"previews/{track_id}.mp3"
        await self._upload_file(
            bucket=self._audio_bucket,
            object_name=key,
            file_path=file_path,
            content_type="audio/mpeg",
        )
        return key

    async def upload_cover(self, track_id: UUID, file_path: Path) -> str:
        key = f"covers/{track_id}.jpg"
        await self._upload_file(
            bucket=self._covers_bucket,
            object_name=key,
            file_path=file_path,
            content_type="image/jpeg",
        )
        return key

    async def _upload_file(self, bucket: str, object_name: str, file_path: Path, content_type: str) -> None:
        if not file_path.exists():
            raise StorageUploadError(f"File for upload not found: {file_path}")

        try:
            logger.debug("Uploading %s to bucket %s (key: %s)", file_path.name, bucket, object_name)
            await asyncio.to_thread(
                self._client.fput_object,
                bucket_name=bucket,
                object_name=object_name,
                file_path=str(file_path),
                content_type=content_type,
            )
        except S3Error as exc:
            raise StorageUploadError(
                f"S3 error uploading {object_name} to bucket {bucket}: {exc.message}"
            ) from exc
        except Exception as exc:
            raise StorageUploadError(
                f"Failed to upload {object_name} to MinIO: {exc}"
            ) from exc