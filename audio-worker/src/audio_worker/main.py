import asyncio
import logging
import signal
import sys

import aio_pika

from audio_worker.config import settings
from audio_worker.infrastructure.broker.consumer import AudioWorkerConsumer
from audio_worker.infrastructure.broker.publisher import RabbitMQEventPublisher
from audio_worker.infrastructure.media import AudioProbe, FFmpegTranscoder
from audio_worker.infrastructure.storage import MinioStorageClient
from audio_worker.service import AudioProcessor


def setup_logging() -> None:
    logging.basicConfig(                                                                                                          
        level=settings.log_level.upper(),                                                                                         
        format="%(asctime)s [%(levelname)s] %(name)s: %(message)s",                                                               
        datefmt="%Y-%m-%d %H:%M:%S",                                                                                              
        stream=sys.stdout,                                                                                                        
    )     


async def main() -> None:
    setup_logging()
    logger = logging.getLogger("audio_worker")
    logger.info("Starting Audio Worker...")

    storage = MinioStorageClient()
    await storage.ensure_buckets()

    logger.info("Connecting to RabbitMQ: %s", settings.rabbitmq_url)
    connection = await aio_pika.connect_robust(settings.rabbitmq_url)

    channel = await connection.channel()
    publisher = RabbitMQEventPublisher(channel)

    prober = AudioProbe()
    transcoder = FFmpegTranscoder()
    processor = AudioProcessor(
        storage=storage, 
        prober=prober, 
        transcoder=transcoder, 
        temp_dir=settings.temp_dir
    )

    consumer = AudioWorkerConsumer(
        connection=connection,
        processor=processor,
        publisher=publisher,
    )
    await consumer.start()

    stop_event = asyncio.Event()
    loop = asyncio.get_running_loop()

    for sig in (signal.SIGINT, signal.SIGTERM):
        loop.add_signal_handler(sig, stop_event.set)
  
    logger.info("Audio Worker has been successfully started and is awaiting tasks. Press Ctrl+C to stop.")
    await stop_event.wait()
  
    logger.info("Termination signal received. Stopping the service...")
    await consumer.close()
    await channel.close()
    await connection.close()
    logger.info("Audio Worker has been shut down correctly.")

if __name__ == "__main__":
    try:
        asyncio.run(main())
    except (KeyboardInterrupt, SystemExit):
        pass