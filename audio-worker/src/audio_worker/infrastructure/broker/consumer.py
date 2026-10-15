import json
import logging
from uuid import UUID

import aio_pika
from aio_pika.abc import AbstractChannel, AbstractConnection, AbstractIncomingMessage

from audio_worker.config import settings
from audio_worker.domain.exceptions import AudioWorkerError, InvalidAudioFormatError
from audio_worker.domain.models import ProcessingFailure, TrackUploadEvent
from audio_worker.service.interfaces import EventPublisher
from audio_worker.service.processor import AudioProcessor

logger = logging.getLogger(__name__)


class AudioWorkerConsumer:

    def __init__(
        self,
        connection: AbstractConnection,
        processor: AudioProcessor,
        publisher: EventPublisher,
    ) -> None:
        self._connection = connection
        self._processor = processor
        self._publisher = publisher
        self._channel: AbstractChannel | None = None

    async def start(self) -> None:
        self._channel = await self._connection.channel()
        await self._channel.set_qos(prefetch_count=1)

        exchange = await self._channel.declare_exchange(settings.rabbitmq_exchange, type="topic", durable=True)
        queue = await self._channel.declare_queue(settings.rabbitmq_queue, durable=True)

        await queue.bind(exchange, routing_key=settings.rabbitmq_routing_key)
        await queue.consume(self._handle_message)

        logger.info(
            "Consumer started: queue=%s, exchange=%s, routing_key=%s",
            settings.rabbitmq_queue,
            settings.rabbitmq_exchange,
            settings.rabbitmq_routing_key,
        )

    async def _handle_message(self, message: AbstractIncomingMessage) -> None:
        track_id: UUID | None = None

        try:
            payload = json.loads(message.body.decode("utf-8"))
            track_id = UUID(payload["track_id"])
            artist_id = UUID(payload["artist_id"])
            raw_key = payload["raw_key"]

            event = TrackUploadEvent(
                track_id=track_id,
                artist_id=artist_id,
                raw_key=raw_key,
            )
        except Exception as exc:
            logger.error("Failed to parse message JSON: %s, body: %s", exc, message.body)
            await message.ack()
            return

        logger.info("[%s] Received audio processing task", track_id)

        try:
            result = await self._processor.process(event)
            await self._publisher.publish_processed(result)
            await message.ack()
            logger.info("[%s] Task successfully completed and acknowledged", track_id)

        except InvalidAudioFormatError as exc:
            logger.warning("[%s] File is corrupted or has an invalid format: %s", track_id, exc)
            await self._publisher.publish_failed(
                ProcessingFailure(track_id=track_id, error_message=str(exc))
            )
            await message.ack()

        except AudioWorkerError as exc:
            logger.error("[%s] Domain error while processing track: %s", track_id, exc)
            await message.nack(requeue=True)

        except Exception as exc:
            logger.exception("[%s] Unexpected system error: %s", track_id, exc)
            await message.nack(requeue=True)

    async def close(self) -> None:
        if self._channel is not None and not self._channel.is_closed:
            await self._channel.close()
            logger.info("Consumer channel closed")