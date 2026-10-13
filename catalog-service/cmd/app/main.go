package main

import (
	"context"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/Lama189/soundwave-platform/catalog-service/internal/config"
	deliveryGrpc "github.com/Lama189/soundwave-platform/catalog-service/internal/delivery/grpc"
	"github.com/Lama189/soundwave-platform/catalog-service/internal/infrastructure/storage"
	"github.com/Lama189/soundwave-platform/catalog-service/internal/repository/postgres"
	"github.com/Lama189/soundwave-platform/catalog-service/internal/service/artist"
	"github.com/Lama189/soundwave-platform/catalog-service/internal/service/track"
	catalogpb "github.com/Lama189/soundwave-platform/gen/go/catalog/v1"
	"google.golang.org/grpc"
)

func main() {
	ctx := context.Background()
	cfg := config.MustLoad()

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

	artistRepo := postgres.NewArtistRepository(pgPool)
	albumRepo := postgres.NewAlbumRepository(pgPool)
	trackRepo := postgres.NewTrackRepository(pgPool)
	trackLikeRepo := postgres.NewTrackLikeRepository(pgPool)
	uowManager := postgres.NewUnitOfWorkManager(pgPool)

	artistService := artist.NewService(artistRepo)
	trackService := track.NewTrackService(uowManager, trackRepo, trackLikeRepo, artistRepo, albumRepo, minioStorage)

	grpcServer := grpc.NewServer()
	artistGrpcHandler := deliveryGrpc.NewArtistGRPCServer(artistService)
	trackGrpcHandler := deliveryGrpc.NewTrackGRPCServer(trackService)

	catalogpb.RegisterArtistServiceServer(grpcServer, artistGrpcHandler)
	catalogpb.RegisterTrackServiceServer(grpcServer, trackGrpcHandler)

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

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down catalog-service gRPC server...")
	grpcServer.GracefulStop()
	log.Println("Server exited properly")
}
