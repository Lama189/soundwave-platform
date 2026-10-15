import json
import logging
from aio_pika import DeliveryMode, Message
from aio_pika.abc import AbstractChannel, AbstractExchange

from audio_worker.config import settings
from audio_worker.domain.models import ProcessingResult, ProcessingFailure

logger = logging.getLogger(__name__)


class RabbitMQEventPublisher:

    def __init__(self, channel: AbstractChannel) -> None:
        self._channel = channel
        self._exchange: AbstractExchange | None = None

    async def _get_exchange(self) -> AbstractExchange:
        if self._exchange is None:
            self._exchange = await self._channel.declare_exchange(
                settings.rabbitmq_exchange, type="topic", durable=True
            )
        return self._exchange

    async def publish_processed(self, result: ProcessingResult) -> None:
        exchange = await self._get_exchange()

        payload = {
            "track_id": str(result.track_id),
            "duration_seconds": result.duration_seconds,
            "file_key": result.file_key,
            "preview_key": result.preview_key,
            "cover_key": result.cover_key,
        }

        body = json.dumps(payload).encode("utf-8")                                                                                
        message = Message(                                                                                                        
            body=body,                                                                                                            
            content_type="application/json",                                                                                      
            delivery_mode=DeliveryMode.PERSISTENT,                                                                                
            message_id=str(result.track_id),                                                                                      
            type="track.processed",                                                                                               
        )      

        routing_key = "track.processed"                                                                                           
        await exchange.publish(message, routing_key=routing_key)                                                                  
        logger.info(                                                                                                              
            "Event %s published to exchange %s (track_id=%s)",                                                              
            routing_key,                                                                                                          
            settings.rabbitmq_exchange,                                                                                           
            result.track_id,                                                                                                      
        )     

    async def publish_failed(self, failure: ProcessingFailure) -> None:                                                           
        exchange = await self._get_exchange()                                                                                     
                                                                                                                                      
        payload = {                                                                                                               
            "track_id": str(failure.track_id),                                                                                    
            "error_message": failure.error_message,                                                                               
        }                                                                                                                         
                                                                                                                                      
        body = json.dumps(payload).encode("utf-8")                                                                                
        message = Message(                                                                                                        
            body=body,                                                                                                            
            content_type="application/json",                                                                                      
            delivery_mode=DeliveryMode.PERSISTENT,                                                                                
            message_id=str(failure.track_id),                                                                                     
            type="track.processing_failed",                                                                                       
        )                                                                                                                         
                                                                                                                                      
        routing_key = "track.processing_failed"                                                                                   
        await exchange.publish(message, routing_key=routing_key)                                                                  
        logger.warning(                                                                                                           
            "Event %s published to exchange %s (track_id=%s, error=%s)",                         
            routing_key,                                                                                                          
            settings.rabbitmq_exchange,                                                                                           
            failure.track_id,                                                                                                     
            failure.error_message,                                                                                                
        )                                                 