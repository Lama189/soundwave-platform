class AudioWorkerError(Exception):
    """Base exception for all audio worker domain errors."""


class InvalidAudioFormatError(AudioWorkerError):
    """Raised when the uploaded file is not recognized as a valid audio format."""


class ProbeError(AudioWorkerError):
    """Raised when ffprobe fails to inspect the audio file."""


class TranscodingError(AudioWorkerError):
    """Raised when ffmpeg fails to transcode or extract audio artifacts."""


class StorageError(AudioWorkerError):
    """Raised when interaction with MinIO/S3 fails."""


class StorageDownloadError(StorageError):
    """Raised when the raw file cannot be downloaded from storage."""


class StorageUploadError(StorageError):
    """Raised when processed audio/cover cannot be uploaded to storage."""
