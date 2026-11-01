package logger

import (
	"context"
	"log/slog"
	"os"
)

type ctxKey string

const requestIDKey ctxKey = "request_id"

func New(env string) *slog.Logger {
	var handler slog.Handler

	switch env {
	case "local", "dev":
		handler = slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelDebug,
		})
	default:
		handler = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelInfo,
		})
	}

	return slog.New(handler)
}

func WIthRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, requestIDKey, requestID)
}

func FromContext(ctx context.Context, log *slog.Logger) *slog.Logger {
	if reqID, ok := ctx.Value(requestIDKey).(string); ok && reqID != "" {
		return log.With(slog.String("request_id", reqID))
	}
	return log
}
