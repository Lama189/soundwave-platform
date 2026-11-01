package contextutil

import (
	"context"

	"github.com/google/uuid"
)

type contextKey string

const (
	ctxUserIDKey   contextKey = "userID"
	ctxUserRoleKey contextKey = "userRole"
)

func WithUserID(ctx context.Context, id uuid.UUID) context.Context {
	return context.WithValue(ctx, ctxUserIDKey, id)
}

func UserIDFromContext(ctx context.Context) (uuid.UUID, bool) {
	id, ok := ctx.Value(ctxUserIDKey).(uuid.UUID)
	return id, ok
}

func WithUserRole(ctx context.Context, role string) context.Context {
	return context.WithValue(ctx, ctxUserRoleKey, role)
}

func UserRoleFromContext(ctx context.Context) (string, bool) {
	role, ok := ctx.Value(ctxUserRoleKey).(string)
	return role, ok
}
