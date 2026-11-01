package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/Lama189/soundwave-platform/billing-service/internal/config"
	deliveryAmqp "github.com/Lama189/soundwave-platform/billing-service/internal/delivery/amqp"
	fxInfra "github.com/Lama189/soundwave-platform/billing-service/internal/infrastructure/fx"
	"github.com/Lama189/soundwave-platform/billing-service/internal/repository/postgres"
	redisRepo "github.com/Lama189/soundwave-platform/billing-service/internal/repository/redis"
	fxService "github.com/Lama189/soundwave-platform/billing-service/internal/service/fx"
	"github.com/Lama189/soundwave-platform/billing-service/internal/service/payout"
	"github.com/Lama189/soundwave-platform/billing-service/internal/service/settlement"
	"github.com/Lama189/soundwave-platform/billing-service/internal/service/wallet"
	amqp "github.com/rabbitmq/amqp091-go"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"
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

	logger.Info("starting billing-service", slog.String("env", cfg.Env))

	// 1. PostgreSQL pool & migrations
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

	// 2. Redis connection & FX cache
	redisClient, err := redisRepo.NewClient(ctx, cfg.Redis)
	if err != nil {
		logger.Error("failed to connect to redis", "error", err)
		os.Exit(1)
	}
	defer redisClient.Close()
	logger.Info("connected to redis", slog.String("addr", fmt.Sprintf("%s:%s", cfg.Redis.Host, cfg.Redis.Port)))

	rateCache := redisRepo.NewRateCacheRepository(redisClient, cfg.Redis.FXTTL)

	// 3. FX provider & service
	fxProvider := fxInfra.NewOpenERAdapter(nil, "")
	currencyConverter := fxService.NewFXService(rateCache, fxProvider, cfg.FX)

	// 4. Repositories & Unit of Work
	uowManager := postgres.NewUnitOfWorkManager(pgPool)
	walletRepo := postgres.NewWalletRepository(pgPool)
	ledgerRepo := postgres.NewLedgerRepository(pgPool)
	payoutRepo := postgres.NewPayoutRepository(pgPool)

	// 5. Domain & Application services
	settlementService := settlement.NewService(uowManager, cfg.Billing.PlatformFeePercent)
	walletService, err := wallet.NewWalletService(uowManager, currencyConverter, walletRepo, ledgerRepo)
	if err != nil {
		logger.Error("failed to init wallet service", "error", err)
		os.Exit(1)
	}
	payoutService := payout.NewPayoutService(uowManager, payoutRepo, walletRepo)

	_ = walletService
	_ = payoutService
	logger.Info("billing settlement, wallet and payout services initialized")

	// 6. RabbitMQ connection & consumers
	amqpConn, err := amqp.Dial(cfg.RabbitMQ.URL)
	if err != nil {
		logger.Error("failed to connect to rabbitmq", "error", err)
		os.Exit(1)
	}
	defer amqpConn.Close()
	logger.Info("connected to rabbitmq")

	orderPaidConsumer := deliveryAmqp.NewOrderPaidConsumer(amqpConn, settlementService, logger)
	if err := orderPaidConsumer.Start(ctx); err != nil {
		logger.Error("failed to start order.paid consumer", "error", err)
		os.Exit(1)
	}

	// 7. gRPC Server
	grpcServer := grpc.NewServer()
	healthServer := health.NewServer()
	grpc_health_v1.RegisterHealthServer(grpcServer, healthServer)
	healthServer.SetServingStatus("", grpc_health_v1.HealthCheckResponse_SERVING)
	reflection.Register(grpcServer)

	lis, err := net.Listen("tcp", cfg.GRPC.Port)
	if err != nil {
		logger.Error("failed to listen on gRPC port", slog.String("port", cfg.GRPC.Port), slog.String("error", err.Error()))
		os.Exit(1)
	}

	go func() {
		logger.Info("starting billing-service gRPC server", slog.String("port", cfg.GRPC.Port))
		if err := grpcServer.Serve(lis); err != nil && !errors.Is(err, grpc.ErrServerStopped) {
			logger.Error("gRPC server error", slog.String("error", err.Error()))
			os.Exit(1)
		}
	}()

	// 8. Graceful shutdown
	<-ctx.Done()

	logger.Info("shutting down billing-service gRPC server...")
	grpcServer.GracefulStop()
	logger.Info("billing-service exited properly")
}
