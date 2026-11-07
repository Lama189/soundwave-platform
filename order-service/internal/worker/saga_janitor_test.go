package worker

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/Lama189/soundwave-platform/order-service/internal/config"
	"github.com/Lama189/soundwave-platform/order-service/internal/domain"
	"github.com/google/uuid"
)

type mockOrderRepo struct {
	timedOutOrders []*domain.Order
	err            error
}

func (m *mockOrderRepo) GetTimedOut(ctx context.Context, now time.Time, limit int) ([]*domain.Order, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.timedOutOrders, nil
}

type mockOrderTimeoutService struct {
	handledIDs []uuid.UUID
	err        error
}

func (m *mockOrderTimeoutService) HandleTimeout(ctx context.Context, orderID uuid.UUID) error {
	if m.err != nil {
		return m.err
	}
	m.handledIDs = append(m.handledIDs, orderID)
	return nil
}

func TestSagaJanitor_ProcessTimedOutOrders(t *testing.T) {
	orderID1 := uuid.New()
	orderID2 := uuid.New()

	order1 := &domain.Order{
		ID: orderID1,
		Saga: domain.SagaState{
			Status: domain.SagaStatusProcessing,
			Step:   domain.StepGrantEntitlements,
		},
	}
	order2 := &domain.Order{
		ID: orderID2,
		Saga: domain.SagaState{
			Status: domain.SagaStatusProcessing,
			Step:   domain.StepSettleBilling,
		},
	}

	repo := &mockOrderRepo{
		timedOutOrders: []*domain.Order{order1, order2},
	}
	svc := &mockOrderTimeoutService{}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	cfg := config.SagaConfig{
		JanitorInterval:  1 * time.Second,
		JanitorBatchSize: 10,
	}

	janitor := NewSagaJanitor(repo, svc, logger, cfg)
	janitor.processTimedOutOrders(context.Background())

	if len(svc.handledIDs) != 2 {
		t.Fatalf("expected 2 handled orders, got %d", len(svc.handledIDs))
	}
	if svc.handledIDs[0] != orderID1 || svc.handledIDs[1] != orderID2 {
		t.Fatalf("unexpected handled ids: %v", svc.handledIDs)
	}
}

func TestSagaJanitor_ProcessTimedOutOrders_Empty(t *testing.T) {
	repo := &mockOrderRepo{
		timedOutOrders: []*domain.Order{},
	}
	svc := &mockOrderTimeoutService{}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	cfg := config.SagaConfig{
		JanitorInterval:  1 * time.Second,
		JanitorBatchSize: 10,
	}

	janitor := NewSagaJanitor(repo, svc, logger, cfg)
	janitor.processTimedOutOrders(context.Background())

	if len(svc.handledIDs) != 0 {
		t.Fatalf("expected 0 handled orders, got %d", len(svc.handledIDs))
	}
}
