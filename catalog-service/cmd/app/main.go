package main

import (
	"context"
	"log"
	"log/slog"
	"net"
	"os/signal"
	"syscall"

	"github.com/Lama189/soundwave-platform/catalog-service/internal/config"
	deliveryAmqp "github.com/Lama189/soundwave-platform/catalog-service/internal/delivery/amqp"
	deliveryGrpc "github.com/Lama189/soundwave-platform/catalog-service/internal/delivery/grpc"
	"github.com/Lama189/soundwave-platform/catalog-service/internal/infrastructure/storage"
	"github.com/Lama189/soundwave-platform/catalog-service/internal/repository/postgres"
	"github.com/Lama189/soundwave-platform/catalog-service/internal/service/album"
	"github.com/Lama189/soundwave-platform/catalog-service/internal/service/artist"
	"github.com/Lama189/soundwave-platform/catalog-service/internal/service/track"
	"github.com/Lama189/soundwave-platform/catalog-service/internal/worker"
	catalogpb "github.com/Lama189/soundwave-platform/gen/go/catalog/v1"
	amqp "github.com/rabbitmq/amqp091-go"
	"google.golang.org/grpc"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	cfg := config.MustLoad()
	logger := slog.Default()

	pgPool, err := postgres.NewPool(ctx, postgres.Config{
		DSN: cfg.Postgres.URL,
	})
	if err != nil {
		log.Fatalf("failed to init postgres: %v", err)
	}
	defer pgPool.Close()

	if err := postgres.RunMigrations(ctx, pgPool); err != nil {
		log.Fatalf("failed to apply migrations: %v", err)
	}

	log.Println("Database connection established and migrations applied successfully")

	minioStorage, err := storage.NewMinioStorage(ctx, cfg.Storage)
	if err != nil {
		log.Fatalf("failed to init minio storage: %v", err)
	}

	amqpConn, err := amqp.Dial(cfg.RabbitMQ.URL)
	if err != nil {
		log.Fatalf("failed to connect to rabbitmq: %v", err)
	}
	defer amqpConn.Close()

	artistRepo := postgres.NewArtistRepository(pgPool)
	albumRepo := postgres.NewAlbumRepository(pgPool)
	trackRepo := postgres.NewTrackRepository(pgPool)
	trackLikeRepo := postgres.NewTrackLikeRepository(pgPool)
	outboxRepo := postgres.NewOutboxRepository(pgPool)
	uowManager := postgres.NewUnitOfWorkManager(pgPool)

	artistService := artist.NewService(artistRepo)
	trackService := track.NewTrackService(uowManager, trackRepo, trackLikeRepo, artistRepo, albumRepo, minioStorage)
	albumService := album.NewService(albumRepo, artistRepo)

	rawAudioConsumer := deliveryAmqp.NewRawAudioConsumer(amqpConn, trackService, logger)
	if err := rawAudioConsumer.Start(ctx); err != nil {
		log.Fatalf("failed to start raw audio consumer: %v", err)
	}

	trackProcessedConsumer := deliveryAmqp.NewTrackProcessedConsumer(amqpConn, trackService, logger)
	if err := trackProcessedConsumer.Start(ctx); err != nil {
		log.Fatalf("failed to start track processed consumer: %v", err)
	}

	outboxRelay := worker.NewOutboxRelay(amqpConn, outboxRepo, logger, cfg.Relay)
	if err := outboxRelay.Start(ctx); err != nil {
		log.Fatalf("failed to start outbox relay: %v", err)
	}

	grpcServer := grpc.NewServer()
	artistGrpcHandler := deliveryGrpc.NewArtistGRPCServer(artistService)
	trackGrpcHandler := deliveryGrpc.NewTrackGRPCServer(trackService)
	albumGRPCHandler := deliveryGrpc.NewAlbumGRPCServer(albumService)

	catalogpb.RegisterArtistServiceServer(grpcServer, artistGrpcHandler)
	catalogpb.RegisterTrackServiceServer(grpcServer, trackGrpcHandler)
	catalogpb.RegisterAlbumServiceServer(grpcServer, albumGRPCHandler)

	lis, err := net.Listen("tcp", cfg.GRPC.Port)
	if err != nil {
		log.Fatalf("failed to listen on %s: %v", cfg.GRPC.Port, err)
	}

	go func() {
		log.Printf("Starting catalog-service gRPC server on %s", cfg.GRPC.Port)
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatalf("gRPC server error: %v", err)
		}
	}()

	<-ctx.Done()

	log.Println("Shutting down catalog-service gRPC server...")
	grpcServer.GracefulStop()
	log.Println("Server exited properly")
}
