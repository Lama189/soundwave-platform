package main

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	paymentpb "github.com/Lama189/soundwave-platform/gen/go/payment/v1"
	"github.com/Lama189/soundwave-platform/payment-service/internal/config"
	deliveryGrpc "github.com/Lama189/soundwave-platform/payment-service/internal/delivery/grpc"
	deliveryHttp "github.com/Lama189/soundwave-platform/payment-service/internal/delivery/http"
	"github.com/Lama189/soundwave-platform/payment-service/internal/domain"
	"github.com/Lama189/soundwave-platform/payment-service/internal/grpcclient"
	"github.com/Lama189/soundwave-platform/payment-service/internal/infrastructure/adapter/mock"
	"github.com/Lama189/soundwave-platform/payment-service/internal/repository/postgres"
	"github.com/Lama189/soundwave-platform/payment-service/internal/service/payment"
	"github.com/Lama189/soundwave-platform/payment-service/internal/worker"
	amqp "github.com/rabbitmq/amqp091-go"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	cfg := config.MustLoad()

	var logHandler slog.Handler
	if cfg.Env == "local" || cfg.Env == "dev" {
		logHandler = slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug})
	} else {
		logHandler = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})
	}
	logger := slog.New(logHandler)
	slog.SetDefault(logger)

	logger.Info("starting payment-service", slog.String("env", cfg.Env))

	pgPool, err := postgres.NewPool(ctx, postgres.Config{
		DSN: cfg.Postgres.URL,
	})
	if err != nil {
		logger.Error("failed to init postgres pool", "error", err)
		os.Exit(1)
	}
	defer pgPool.Close()

	if err := postgres.RunMigrations(ctx, pgPool); err != nil {
		logger.Error("failed to apply migrations", "error", err)
		os.Exit(1)
	}
	logger.Info("database migrations applied successfully")

	amqpConn, err := amqp.Dial(cfg.RabbitMQ.URL)
	if err != nil {
		logger.Error("failed to connect to rabbitmq", "error", err)
		os.Exit(1)
	}
	defer amqpConn.Close()

	orderConn, err := grpc.NewClient(
		cfg.Order.URL,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		logger.Error("failed to connect to order-service", "url", cfg.Order.URL, "error", err)
		os.Exit(1)
	}
	defer orderConn.Close()

	orderClient := grpcclient.NewOrderClient(orderConn, cfg.Order.Timeout)

	mockGateway := mock.NewAdapter(cfg.MockAdapter)
	gateways := map[domain.PaymentProvider]payment.PaymentGateway{
		domain.PaymentProviderMock: mockGateway,
	}

	uowManager := postgres.NewUnitOfWorkManager(pgPool)
	paymentService := payment.NewService(uowManager, orderClient, gateways)

	outboxRepo := postgres.NewOutboxRepository(pgPool)
	outboxRelay := worker.NewOutboxRelay(amqpConn, outboxRepo, logger, cfg.Relay)
	if err := outboxRelay.Start(ctx); err != nil {
		logger.Error("failed to start outbox relay", "error", err)
		os.Exit(1)
	}
	logger.Info("outbox relay worker started successfully")

	grpcServer := grpc.NewServer()
	paymentGrpcHandler := deliveryGrpc.NewPaymentGRPCServer(paymentService)
	paymentpb.RegisterPaymentServiceServer(grpcServer, paymentGrpcHandler)

	grpcLis, err := net.Listen("tcp", cfg.GRPC.Port)
	if err != nil {
		logger.Error("failed to listen on gRPC port", "port", cfg.GRPC.Port, "error", err)
		os.Exit(1)
	}

	go func() {
		logger.Info("payment-service gRPC server listening", "port", cfg.GRPC.Port)
		if err := grpcServer.Serve(grpcLis); err != nil && !errors.Is(err, grpc.ErrServerStopped) {
			logger.Error("gRPC server failed", "error", err)
			stop()
		}
	}()

	webhookHandler := deliveryHttp.NewWebhookHandler(paymentService, cfg.Webhook.MaxWebhookBodySize)
	httpRouter := deliveryHttp.NewRouter(webhookHandler)

	httpServer := &http.Server{
		Addr:         cfg.HTTP.Port,
		Handler:      httpRouter,
		ReadTimeout:  cfg.HTTP.ReadTimeout,
		WriteTimeout: cfg.HTTP.WriteTimeout,
		IdleTimeout:  cfg.HTTP.IdleTimeout,
	}

	go func() {
		logger.Info("payment-service HTTP server listening", "port", cfg.HTTP.Port)
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("HTTP server failed", "error", err)
			stop()
		}
	}()

	<-ctx.Done()
	logger.Info("shutting down payment-service...")

	shutdownCtx, cancelShutdown := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelShutdown()

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		logger.Error("HTTP server forced to shutdown", "error", err)
	}

	grpcServer.GracefulStop()
	logger.Info("payment-service stopped gracefully")
}
