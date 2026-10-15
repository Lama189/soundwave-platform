import asyncio
import logging
from pathlib import Path

from audio_worker.config import settings
from audio_worker.domain.exceptions import TranscodingError
from audio_worker.domain.models import AudioMetadata, TranscodedMedia

logger = logging.getLogger(__name__)


class FFmpegTranscoder:
    """Транскодер аудиофайлов на базе системной утилиты ffmpeg."""

    async def transcode(
        self,
        input_path: Path,
        output_dir: Path,
        metadata: AudioMetadata,
    ) -> TranscodedMedia:
        """Выполняет полный цикл конвертации: основной трек, 30s превью и обложка."""
        output_dir.mkdir(parents=True, exist_ok=True)

        main_track_path = output_dir / "track.mp3"
        preview_path = output_dir / "preview.mp3"
        cover_path = output_dir / "cover.jpg" if metadata.has_embedded_cover else None

        # 1. Конвертация основного трека в нормализованный MP3
        await self._convert_to_mp3(input_path, main_track_path)

        # 2. Создание 30-секундного превью с fade in/out
        await self._create_preview(input_path, preview_path, metadata.duration_seconds)

        # 3. Извлечение обложки, если она есть
        actual_cover_path: Path | None = None
        if cover_path is not None:
            actual_cover_path = await self._extract_cover(input_path, cover_path)

        return TranscodedMedia(
            audio_file_path=main_track_path,
            preview_file_path=preview_path,
            cover_file_path=actual_cover_path,
            duration_seconds=metadata.duration_seconds,
        )

    async def _convert_to_mp3(self, input_path: Path, output_path: Path) -> None:
        """Перекодирует аудио в стандартизированный MP3 (320 kbps, 44100 Hz)."""
        cmd = [
            "ffmpeg",
            "-y",
            "-i", str(input_path),
            "-vn",  # отбрасываем видео/обложку из аудиофайла
            "-codec:a", "libmp3lame",
            "-b:a", f"{settings.audio_bitrate_kbps}k",
            "-ar", str(settings.audio_sample_rate_hz),
            str(output_path),
        ]
        await self._run_command(cmd, "конвертация в основной MP3")

    async def _create_preview(self, input_path: Path, output_path: Path, total_duration: int) -> None:
        """Создает 30-секундное превью трека с аудио-эффектом затухания."""
        preview_len = min(settings.preview_duration_seconds, total_duration)

        # Если трек длинный (> 45s), начинаем превью с 15-й секунды для лучшего момента
        start_offset = 15 if total_duration > 45 else 0

        # Эффекты: fade-in 1 сек в начале, fade-out 2 сек в конце
        fade_out_start = max(0, preview_len - 2)
        audio_filter = f"afade=t=in:st=0:d=1,afade=t=out:st={fade_out_start}:d=2"

        cmd = [
            "ffmpeg",
            "-y",
            "-ss", str(start_offset),
            "-t", str(preview_len),
            "-i", str(input_path),
            "-vn",
            "-codec:a", "libmp3lame",
            "-b:a", "192k",
            "-ar", str(settings.audio_sample_rate_hz),
            "-af", audio_filter,
            str(output_path),
        ]
        await self._run_command(cmd, "создание 30s превью")

    async def _extract_cover(self, input_path: Path, output_path: Path) -> Path | None:
        """Извлекает обложку (attached picture) без перекодирования."""
        cmd = [
            "ffmpeg",
            "-y",
            "-i", str(input_path),
            "-an",  # отбрасываем аудио
            "-vcodec", "copy",
            str(output_path),
        ]
        try:
            await self._run_command(cmd, "извлечение обложки")
            if output_path.exists() and output_path.stat().st_size > 0:
                return output_path
        except TranscodingError as exc:
            logger.warning("Не удалось извлечь обложку (пропускаем): %s", exc)

        return None

    async def _run_command(self, cmd: list[str], description: str) -> None:
        """Запускает процесс ffmpeg с контролем ошибок."""
        try:
            process = await asyncio.create_subprocess_exec(
                *cmd,
                stdout=asyncio.subprocess.PIPE,
                stderr=asyncio.subprocess.PIPE,
            )
            _, stderr = await process.communicate()
        except Exception as exc:
            raise TranscodingError(f"Ошибка запуска ffmpeg ({description}): {exc}") from exc

        if process.returncode != 0:
            error_msg = stderr.decode(errors="replace").strip()
            raise TranscodingError(f"ffmpeg завершился с ошибкой при '{description}': {error_msg}")
