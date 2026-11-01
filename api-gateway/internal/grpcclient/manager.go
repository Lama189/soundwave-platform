package grpcclient

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	chiMiddleware "github.com/go-chi/chi/v5/middleware"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/metadata"

	"github.com/Lama189/soundwave-platform/api-gateway/pkg/contextutil"
)

type Config struct {
	Timeout time.Duration
}

type Manager struct {
	logger *slog.Logger
	mu     sync.RWMutex
	conns  map[string]*grpc.ClientConn
}

func NewManager(logger *slog.Logger) *Manager {
	return &Manager{
		logger: logger,
		conns:  make(map[string]*grpc.ClientConn),
	}
}

func (m *Manager) GetConn(ctx context.Context, target string, extraOpts ...grpc.DialOption) (*grpc.ClientConn, error) {
	m.mu.RLock()
	conn, exists := m.conns[target]
	m.mu.RUnlock()
	if exists {
		return conn, nil
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if conn, exists := m.conns[target]; exists {
		return conn, nil
	}

	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithKeepaliveParams(keepalive.ClientParameters{
			Time:                10 * time.Second,
			Timeout:             3 * time.Second,
			PermitWithoutStream: true,
		}),
		grpc.WithChainUnaryInterceptor(
			m.metadataPropagationInterceptor(),
			m.loggingInterceptor(),
		),
	}

	opts = append(opts, extraOpts...)

	newConn, err := grpc.NewClient(target, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create gRPC client for %s: %w", target, err)
	}

	m.conns[target] = newConn
	m.logger.Info("gRPC connection initialized", slog.String("target", target))

	return newConn, err
}

func (m *Manager) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	var firstErr error
	for target, conn := range m.conns {
		if err := conn.Close(); err != nil {
			m.logger.Error("failed to close gRPC connection", slog.String("target", target))
			if firstErr == nil {
				firstErr = err
			}
		} else {
			m.logger.Info("gRPC connection closed", slog.String("target", target))
		}
	}

	clear(m.conns)
	return firstErr
}

func (m *Manager) metadataPropagationInterceptor() grpc.UnaryClientInterceptor {
	return func(
		ctx context.Context,
		method string,
		req, reply any,
		cc *grpc.ClientConn,
		invoker grpc.UnaryInvoker,
		opts ...grpc.CallOption,
	) error {
		md := metadata.MD{}

		reqID := chiMiddleware.GetReqID(ctx)
		if reqID == "" {
			if v, ok := ctx.Value("requestID").(string); ok {
				reqID = v
			}
		}
		if reqID != "" {
			md.Set("x-request-id", reqID)
		}

		if userID, ok := contextutil.UserIDFromContext(ctx); ok {
			md.Set("x-user-id", userID.String())
		}
		if role, ok := contextutil.UserRoleFromContext(ctx); ok && role != "" {
			md.Set("x-user-role", role)
		}

		if md.Len() > 0 {
			ctx = metadata.NewOutgoingContext(ctx, md)
		}

		return invoker(ctx, method, req, reply, cc, opts...)
	}
}

func (m *Manager) loggingInterceptor() grpc.UnaryClientInterceptor {
	return func(
		ctx context.Context,
		method string,
		req, reply any,
		cc *grpc.ClientConn,
		invoker grpc.UnaryInvoker,
		opts ...grpc.CallOption,
	) error {
		start := time.Now().UTC()
		err := invoker(ctx, method, req, reply, cc, opts...)
		duration := time.Since(start)

		if err != nil {
			m.logger.Error(
				"gRPC call failed",
				slog.String("method", method),
				slog.Duration("duration", duration),
				slog.String("error", err.Error()),
			)
		} else {
			m.logger.Debug(
				"gRPC call succeeded",
				slog.String("method", method),
				slog.Duration("duration", duration),
			)
		}

		return err
	}
}
