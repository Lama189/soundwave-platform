package settlement_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/Lama189/soundwave-platform/billing-service/internal/domain"
	"github.com/Lama189/soundwave-platform/billing-service/internal/service/settlement"
	"github.com/Lama189/soundwave-platform/billing-service/internal/uow"
	"github.com/Lama189/soundwave-platform/pkg/itemtype"
	"github.com/google/uuid"
)

type mockUoWManager struct {
	work uow.UnitOfWork
}

func (m *mockUoWManager) Do(ctx context.Context, fn func(work uow.UnitOfWork) error) error {
	return fn(m.work)
}

type mockUnitOfWork struct {
	walletRepo            uow.WalletRepository
	ledgerRepo            uow.LedgerRepository
	outboxRepo            uow.OutboxRepository
	processedCommandsRepo uow.ProcessedCommandRepository
}

func (m *mockUnitOfWork) Wallets() uow.WalletRepository                  { return m.walletRepo }
func (m *mockUnitOfWork) Ledger() uow.LedgerRepository                   { return m.ledgerRepo }
func (m *mockUnitOfWork) Payouts() uow.PayoutRepository                  { return nil }
func (m *mockUnitOfWork) Outbox() uow.OutboxRepository                   { return m.outboxRepo }
func (m *mockUnitOfWork) ProcessedEvents() uow.ProcessedEventsRepository { return nil }
func (m *mockUnitOfWork) ProcessedCommands() uow.ProcessedCommandRepository {
	return m.processedCommandsRepo
}

type mockWalletRepo struct {
	createFn func(ctx context.Context, wallet *domain.Wallet) error
	getForUp func(ctx context.Context, artistID uuid.UUID) (*domain.Wallet, error)
	updateFn func(ctx context.Context, wallet *domain.Wallet) error
}

func (m *mockWalletRepo) Create(ctx context.Context, wallet *domain.Wallet) error {
	if m.createFn != nil {
		return m.createFn(ctx, wallet)
	}
	return nil
}

func (m *mockWalletRepo) GetByArtistID(ctx context.Context, artistID uuid.UUID) (*domain.Wallet, error) {
	return nil, nil
}

func (m *mockWalletRepo) GetByArtistIDForUpdate(ctx context.Context, artistID uuid.UUID) (*domain.Wallet, error) {
	if m.getForUp != nil {
		return m.getForUp(ctx, artistID)
	}
	return nil, domain.ErrNotFound
}

func (m *mockWalletRepo) Update(ctx context.Context, wallet *domain.Wallet) error {
	if m.updateFn != nil {
		return m.updateFn(ctx, wallet)
	}
	return nil
}

type mockLedgerRepo struct {
	appendFn func(ctx context.Context, entry *domain.LedgerEntry) error
}

func (m *mockLedgerRepo) Append(ctx context.Context, entry *domain.LedgerEntry) error {
	if m.appendFn != nil {
		return m.appendFn(ctx, entry)
	}
	return nil
}

func (m *mockLedgerRepo) GetByWalletID(ctx context.Context, walletID uuid.UUID, limit, offset int) ([]*domain.LedgerEntry, error) {
	return nil, nil
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

func TestProcessSettleOrder_Success(t *testing.T) {
	commandID := uuid.New()
	sagaID := uuid.New()
	orderID := uuid.New()
	buyerID := uuid.New()
	artistID := uuid.New()
	itemID := uuid.New()

	existingWallet, _ := domain.NewWallet(artistID)

	var updatedWallet *domain.Wallet
	var appendedEntries []*domain.LedgerEntry
	var capturedEvent *domain.Event
	var markedCommandID, markedSagaID uuid.UUID
	var markedCommandName string

	walletRepo := &mockWalletRepo{
		getForUp: func(ctx context.Context, aid uuid.UUID) (*domain.Wallet, error) {
			return existingWallet, nil
		},
		updateFn: func(ctx context.Context, w *domain.Wallet) error {
			updatedWallet = w
			return nil
		},
	}

	ledgerRepo := &mockLedgerRepo{
		appendFn: func(ctx context.Context, entry *domain.LedgerEntry) error {
			appendedEntries = append(appendedEntries, entry)
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
		walletRepo:            walletRepo,
		ledgerRepo:            ledgerRepo,
		outboxRepo:            outboxRepo,
		processedCommandsRepo: inboxRepo,
	}
	uowMgr := &mockUoWManager{work: uowWork}

	svc := settlement.NewService(uowMgr, 15)

	input := settlement.SettleOrderInput{
		CommandID:   commandID,
		SagaID:      sagaID,
		OrderID:     orderID,
		BuyerUserID: buyerID,
		TotalAmount: 1000,
		Currency:    domain.CurrencyUSD,
		PaymentID:   uuid.New(),
		PaidAt:      time.Now().UTC(),
		Items: []settlement.SettleOrderItemInput{
			{
				ItemID:   itemID,
				ItemType: itemtype.Track,
				ArtistID: artistID,
				Price:    1000,
				Currency: domain.CurrencyUSD,
			},
		},
	}

	if err := svc.ProcessSettleOrder(context.Background(), input); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if updatedWallet == nil {
		t.Fatal("expected wallet to be updated")
	}
	if updatedWallet.Balance.Amount != 850 {
		t.Errorf("expected wallet balance 850, got %d", updatedWallet.Balance.Amount)
	}

	if len(appendedEntries) != 1 {
		t.Fatalf("expected 1 ledger entry, got %d", len(appendedEntries))
	}

	if capturedEvent == nil {
		t.Fatal("expected outbox event to be recorded")
	}
	if capturedEvent.EventType != domain.EventTypeSagaReplyBillingSettled {
		t.Errorf("expected event type %s, got %s", domain.EventTypeSagaReplyBillingSettled, capturedEvent.EventType)
	}

	var payload settlement.BillingReplyPayload
	if err := json.Unmarshal(capturedEvent.Payload, &payload); err != nil {
		t.Fatalf("failed to unmarshal payload: %v", err)
	}
	if !payload.Success {
		t.Error("expected success to be true")
	}
	if payload.OrderID != orderID {
		t.Errorf("expected order id %s, got %s", orderID, payload.OrderID)
	}
	if payload.SagaID != sagaID {
		t.Errorf("expected saga id %s, got %s", sagaID, payload.SagaID)
	}

	if markedCommandID != commandID || markedSagaID != sagaID || markedCommandName != "billing.settle_order" {
		t.Errorf("unexpected inbox mark: cmd=%s, saga=%s, name=%s", markedCommandID, markedSagaID, markedCommandName)
	}
}

func TestProcessSettleOrder_IdempotencyFastPath(t *testing.T) {
	commandID := uuid.New()
	walletUpdated := false
	outboxRecorded := false

	inboxRepo := &mockInboxRepo{
		hasBeenProcessedFn: func(ctx context.Context, cid uuid.UUID) (bool, error) {
			return true, nil
		},
	}
	walletRepo := &mockWalletRepo{
		updateFn: func(ctx context.Context, w *domain.Wallet) error {
			walletUpdated = true
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
		walletRepo:            walletRepo,
		outboxRepo:            outboxRepo,
		processedCommandsRepo: inboxRepo,
	}
	uowMgr := &mockUoWManager{work: uowWork}

	svc := settlement.NewService(uowMgr, 15)

	input := settlement.SettleOrderInput{
		CommandID: commandID,
		SagaID:    uuid.New(),
		OrderID:   uuid.New(),
		Items: []settlement.SettleOrderItemInput{
			{
				ItemID:   uuid.New(),
				ItemType: itemtype.Track,
				ArtistID: uuid.New(),
				Price:    100,
				Currency: domain.CurrencyUSD,
			},
		},
	}

	if err := svc.ProcessSettleOrder(context.Background(), input); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if walletUpdated {
		t.Error("expected wallet not to be updated when command already processed")
	}
	if outboxRecorded {
		t.Error("expected outbox event not to be recorded when command already processed")
	}
}

func TestProcessSettleOrder_ValidationError(t *testing.T) {
	svc := settlement.NewService(nil, 15)

	if err := svc.ProcessSettleOrder(context.Background(), settlement.SettleOrderInput{}); err == nil {
		t.Fatal("expected error on invalid input")
	}
}
