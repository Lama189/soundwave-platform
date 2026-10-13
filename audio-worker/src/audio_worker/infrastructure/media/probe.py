import asyncio
import json
import logging
from pathlib import Path

from audio_worker.domain.exceptions import InvalidAudioFormatError, ProbeError
from audio_worker.domain.models import AudioMetadata

logger = logging.getLogger(__name__)


class AudioProbe:
    """Анализатор медиафайлов с использованием ffprobe."""

    async def probe(self, file_path: Path) -> AudioMetadata:
        """Запускает ffprobe и возвращает метаданные аудиофайла."""
        if not file_path.exists():
            raise ProbeError(f"Файл для анализа не найден: {file_path}")

        cmd = [
            "ffprobe",
            "-v", "quiet",
            "-print_format", "json",
            "-show_format",
            "-show_streams",
            str(file_path),
        ]

        try:
            process = await asyncio.create_subprocess_exec(
                *cmd,
                stdout=asyncio.subprocess.PIPE,
                stderr=asyncio.subprocess.PIPE,
            )
            stdout, stderr = await process.communicate()
        except Exception as exc:
            raise ProbeError(f"Не удалось запустить ffprobe: {exc}") from exc

        if process.returncode != 0:
            error_msg = stderr.decode(errors="replace").strip()
            raise InvalidAudioFormatError(f"ffprobe не смог прочитать файл (код {process.returncode}): {error_msg}")

        try:
            data = json.loads(stdout.decode(errors="replace"))
        except json.JSONDecodeError as exc:
            raise ProbeError("ffprobe вернул некорректный JSON") from exc

        streams = data.get("streams", [])
        format_info = data.get("format", {})

        # Ищем аудиопоток
        audio_stream = next((s for s in streams if s.get("codec_type") == "audio"), None)
        if not audio_stream:
            raise InvalidAudioFormatError("В загруженном файле отсутствует аудиопоток")

        # Проверяем наличие встроенного изображения (обложки)
        has_embedded_cover = any(
            s.get("codec_type") == "video" and s.get("disposition", {}).get("attached_pic") == 1
            for s in streams
        )

        # Вычисляем длительность в секундах
        raw_duration = format_info.get("duration") or audio_stream.get("duration")
        if not raw_duration:
            raise InvalidAudioFormatError("Не удалось определить длительность аудиофайла")

        try:
            duration_seconds = max(1, round(float(raw_duration)))
        except (ValueError, TypeError) as exc:
            raise InvalidAudioFormatError(f"Некорректное значение длительности: {raw_duration}") from exc

        # Дополнительные метаданные
        bit_rate = int(format_info["bit_rate"]) if "bit_rate" in format_info else None
        sample_rate = int(audio_stream["sample_rate"]) if "sample_rate" in audio_stream else None
        channels = int(audio_stream["channels"]) if "channels" in audio_stream else None

        logger.debug(
            "ffprobe успешно проанализировал %s: duration=%ds, cover=%s",
            file_path.name,
            duration_seconds,
            has_embedded_cover,
        )

        return AudioMetadata(
            duration_seconds=duration_seconds,
            format_name=format_info.get("format_name", "unknown"),
            bit_rate=bit_rate,
            sample_rate=sample_rate,
            channels=channels,
            has_embedded_cover=has_embedded_cover,
        )
