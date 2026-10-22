package main

import (
	"context"
	"errors"
	"log"
	"log/slog"
	"net"
	"os/signal"
	"syscall"

	orderpb "github.com/Lama189/soundwave-platform/gen/go/order/v1"
	"github.com/Lama189/soundwave-platform/order-service/internal/config"
	deliveryGrpc "github.com/Lama189/soundwave-platform/order-service/internal/delivery/grpc"
	"github.com/Lama189/soundwave-platform/order-service/internal/grpcclient"
	"github.com/Lama189/soundwave-platform/order-service/internal/repository/postgres"
	"github.com/Lama189/soundwave-platform/order-service/internal/service/order"
	"github.com/Lama189/soundwave-platform/order-service/internal/worker"
	amqp "github.com/rabbitmq/amqp091-go"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
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

	amqpConn, err := amqp.Dial(cfg.RabbitMQ.URL)
	if err != nil {
		log.Fatalf("failed to connect to rabbitmq: %v", err)
	}
	defer amqpConn.Close()

	orderRepo := postgres.NewOrderRepository(pgPool)
	outboxRepo := postgres.NewOutboxRepository(pgPool)
	uowManager := postgres.NewUnitOfWorkManager(pgPool)

	catalogConn, err := grpc.NewClient(
		cfg.Catalog.URL,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		log.Fatalf("failed to connect to catalog-service at %s: %v", cfg.Catalog.URL, err)
	}
	defer catalogConn.Close()

	catalogClient := grpcclient.NewCatalogClient(catalogConn, cfg.Catalog.Timeout)

	orderService := order.NewOrderService(uowManager, orderRepo, catalogClient)

	outboxRelay := worker.NewOutboxRelay(amqpConn, outboxRepo, logger, cfg.Relay)
	if err := outboxRelay.Start(ctx); err != nil {
		log.Fatalf("failed to start outbox relay: %v", err)
	}

	grpcServer := grpc.NewServer()
	orderGrpcHandler := deliveryGrpc.NewOrderGRPCServer(orderService)
	orderpb.RegisterOrderServiceServer(grpcServer, orderGrpcHandler)

	lis, err := net.Listen("tcp", cfg.GRPC.Port)
	if err != nil {
		log.Fatalf("failed to listen on %s: %v", cfg.GRPC.Port, err)
	}

	go func() {
		log.Printf("Starting order-service gRPC server on %s", cfg.GRPC.Port)
		if err := grpcServer.Serve(lis); err != nil && !errors.Is(err, grpc.ErrServerStopped) {
			log.Fatalf("gRPC server error: %v", err)
		}
	}()

	<-ctx.Done()

	log.Println("Shutting down order-service gRPC server...")
	grpcServer.GracefulStop()
	log.Println("order-service exited properly")
}
