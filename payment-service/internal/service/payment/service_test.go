package payment_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/Lama189/soundwave-platform/payment-service/internal/domain"
	"github.com/Lama189/soundwave-platform/payment-service/internal/service/payment"
	"github.com/Lama189/soundwave-platform/payment-service/internal/uow"
	"github.com/google/uuid"
)

type mockUoWManager struct {
	work uow.UnitOfWork
}

func (m *mockUoWManager) Do(ctx context.Context, fn func(work uow.UnitOfWork) error) error {
	return fn(m.work)
}

type mockUnitOfWork struct {
	paymentRepo           uow.PaymentRepository
	outboxRepo            uow.OutboxRepository
	processedCommandsRepo uow.ProcessedCommandRepository
}

func (m *mockUnitOfWork) Payments() uow.PaymentRepository           { return m.paymentRepo }
func (m *mockUnitOfWork) WebhookEvents() uow.WebhookEventRepository { return nil }
func (m *mockUnitOfWork) Outbox() uow.OutboxRepository              { return m.outboxRepo }
func (m *mockUnitOfWork) ProcessedCommands() uow.ProcessedCommandRepository {
	return m.processedCommandsRepo
}

type mockPaymentRepo struct {
	getByOrderIDFn func(ctx context.Context, orderID uuid.UUID) (*domain.Payment, error)
	updateFn       func(ctx context.Context, payment *domain.Payment) error
}

func (m *mockPaymentRepo) Create(ctx context.Context, p *domain.Payment) error { return nil }
func (m *mockPaymentRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Payment, error) {
	return nil, nil
}
func (m *mockPaymentRepo) GetByIDForUpdate(ctx context.Context, id uuid.UUID) (*domain.Payment, error) {
	return nil, nil
}
func (m *mockPaymentRepo) GetByExternalID(ctx context.Context, externalID string) (*domain.Payment, error) {
	return nil, nil
}
func (m *mockPaymentRepo) GetByExternalIDForUpdate(ctx context.Context, externalID string) (*domain.Payment, error) {
	return nil, nil
}
func (m *mockPaymentRepo) GetByIdempotencyKey(ctx context.Context, key string) (*domain.Payment, error) {
	return nil, nil
}
func (m *mockPaymentRepo) GetByOrderID(ctx context.Context, orderID uuid.UUID) (*domain.Payment, error) {
	if m.getByOrderIDFn != nil {
		return m.getByOrderIDFn(ctx, orderID)
	}
	return nil, domain.ErrNotFound
}
func (m *mockPaymentRepo) Update(ctx context.Context, p *domain.Payment) error {
	if m.updateFn != nil {
		return m.updateFn(ctx, p)
	}
	return nil
}

type mockOutboxRepo struct {
	recordFn func(ctx context.Context, event *domain.Event) error
}

func (m *mockOutboxRepo) Record(ctx context.Context, event *domain.Event) error {
	if m.recordFn != nil {
		return m.recordFn(ctx, event)
	}
	return nil
}

type mockInboxRepo struct {
	hasBeenProcessedFn func(ctx context.Context, commandID uuid.UUID) (bool, error)
	markProcessedFn    func(ctx context.Context, commandID, sagaID uuid.UUID, commandName string) error
}

func (m *mockInboxRepo) HasBeenProcessed(ctx context.Context, commandID uuid.UUID) (bool, error) {
	if m.hasBeenProcessedFn != nil {
		return m.hasBeenProcessedFn(ctx, commandID)
	}
	return false, nil
}

func (m *mockInboxRepo) MarkProcessed(ctx context.Context, commandID, sagaID uuid.UUID, commandName string) error {
	if m.markProcessedFn != nil {
		return m.markProcessedFn(ctx, commandID, sagaID, commandName)
	}
	return nil
}

func TestProcessRefundCommand_Success(t *testing.T) {
	orderID := uuid.New()
	commandID := uuid.New()
	sagaID := uuid.New()
	paymentID := uuid.New()

	existingPayment, err := domain.RestorePayment(
		paymentID,
		orderID,
		uuid.New(),
		1000,
		domain.CurrencyUSD,
		domain.PaymentStatusSucceeded,
		domain.PaymentProviderMock,
		"ext_123",
		"sec_123",
		"idem_123",
		nil,
		time.Now().UTC(),
		time.Now().UTC(),
	)
	if err != nil {
		t.Fatalf("unexpected error on RestorePayment: %v", err)
	}

	var updatedPayment *domain.Payment
	var capturedEvent *domain.Event
	var markedCommandID, markedSagaID uuid.UUID
	var markedCommandName string

	paymentRepo := &mockPaymentRepo{
		getByOrderIDFn: func(ctx context.Context, oid uuid.UUID) (*domain.Payment, error) {
			return existingPayment, nil
		},
		updateFn: func(ctx context.Context, p *domain.Payment) error {
			updatedPayment = p
			return nil
		},
	}

	outboxRepo := &mockOutboxRepo{
		recordFn: func(ctx context.Context, event *domain.Event) error {
			capturedEvent = event
			return nil
		},
	}

	inboxRepo := &mockInboxRepo{
		hasBeenProcessedFn: func(ctx context.Context, cid uuid.UUID) (bool, error) {
			return false, nil
		},
		markProcessedFn: func(ctx context.Context, cid, sid uuid.UUID, cname string) error {
			markedCommandID = cid
			markedSagaID = sid
			markedCommandName = cname
			return nil
		},
	}

	uowWork := &mockUnitOfWork{
		paymentRepo:           paymentRepo,
		outboxRepo:            outboxRepo,
		processedCommandsRepo: inboxRepo,
	}
	uowMgr := &mockUoWManager{work: uowWork}

	svc := payment.NewService(uowMgr, nil, nil)

	input := payment.RefundCommandInput{
		CommandID: commandID,
		SagaID:    sagaID,
		OrderID:   orderID,
		Reason:    "settlement failed",
	}

	if err := svc.ProcessRefundCommand(context.Background(), input); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if updatedPayment == nil {
		t.Fatal("expected payment to be updated")
	}
	if updatedPayment.Status != domain.PaymentStatusRefunded {
		t.Errorf("expected status %s, got %s", domain.PaymentStatusRefunded, updatedPayment.Status)
	}

	if capturedEvent == nil {
		t.Fatal("expected outbox event to be recorded")
	}
	if capturedEvent.EventType != domain.EventTypeSagaReplyPaymentRefunded {
		t.Errorf("expected event type %s, got %s", domain.EventTypeSagaReplyPaymentRefunded, capturedEvent.EventType)
	}

	var payload domain.PaymentSagaReplyPayload
	if err := json.Unmarshal(capturedEvent.Payload, &payload); err != nil {
		t.Fatalf("failed to unmarshal payload: %v", err)
	}
	if !payload.Success {
		t.Error("expected success true")
	}
	if payload.OrderID != orderID {
		t.Errorf("expected order id %s, got %s", orderID, payload.OrderID)
	}
	if payload.SagaID != sagaID {
		t.Errorf("expected saga id %s, got %s", sagaID, payload.SagaID)
	}

	if markedCommandID != commandID || markedSagaID != sagaID || markedCommandName != "payment.refund" {
		t.Errorf("unexpected inbox mark: cmd=%s, saga=%s, name=%s", markedCommandID, markedSagaID, markedCommandName)
	}
}

func TestProcessRefundCommand_IdempotencyFastPath(t *testing.T) {
	commandID := uuid.New()
	paymentUpdated := false
	outboxRecorded := false

	inboxRepo := &mockInboxRepo{
		hasBeenProcessedFn: func(ctx context.Context, cid uuid.UUID) (bool, error) {
			return true, nil
		},
	}
	paymentRepo := &mockPaymentRepo{
		updateFn: func(ctx context.Context, p *domain.Payment) error {
			paymentUpdated = true
			return nil
		},
	}
	outboxRepo := &mockOutboxRepo{
		recordFn: func(ctx context.Context, event *domain.Event) error {
			outboxRecorded = true
			return nil
		},
	}

	uowWork := &mockUnitOfWork{
		paymentRepo:           paymentRepo,
		outboxRepo:            outboxRepo,
		processedCommandsRepo: inboxRepo,
	}
	uowMgr := &mockUoWManager{work: uowWork}

	svc := payment.NewService(uowMgr, nil, nil)

	input := payment.RefundCommandInput{
		CommandID: commandID,
		SagaID:    uuid.New(),
		OrderID:   uuid.New(),
	}

	if err := svc.ProcessRefundCommand(context.Background(), input); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if paymentUpdated {
		t.Error("expected payment not to be updated when command already processed")
	}
	if outboxRecorded {
		t.Error("expected outbox event not to be recorded when command already processed")
	}
}

func TestProcessRefundCommand_PaymentNotFound(t *testing.T) {
	paymentRepo := &mockPaymentRepo{
		getByOrderIDFn: func(ctx context.Context, oid uuid.UUID) (*domain.Payment, error) {
			return nil, domain.ErrNotFound
		},
	}
	inboxRepo := &mockInboxRepo{
		hasBeenProcessedFn: func(ctx context.Context, cid uuid.UUID) (bool, error) {
			return false, nil
		},
	}

	uowWork := &mockUnitOfWork{
		paymentRepo:           paymentRepo,
		processedCommandsRepo: inboxRepo,
	}
	uowMgr := &mockUoWManager{work: uowWork}

	svc := payment.NewService(uowMgr, nil, nil)

	input := payment.RefundCommandInput{
		CommandID: uuid.New(),
		SagaID:    uuid.New(),
		OrderID:   uuid.New(),
	}

	if err := svc.ProcessRefundCommand(context.Background(), input); err == nil {
		t.Fatal("expected error when payment not found")
	}
}

func TestProcessRefundCommand_ValidationError(t *testing.T) {
	svc := payment.NewService(nil, nil, nil)

	if err := svc.ProcessRefundCommand(context.Background(), payment.RefundCommandInput{}); err == nil {
		t.Fatal("expected error on empty input")
	}
}
