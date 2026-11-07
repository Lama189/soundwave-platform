package user_purchase_test

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/Lama189/soundwave-platform/catalog-service/internal/domain"
	"github.com/Lama189/soundwave-platform/catalog-service/internal/service/user_purchase"
	"github.com/Lama189/soundwave-platform/catalog-service/internal/uow"
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
	tracksRepo            uow.TrackRepository
	purchasesRepo         uow.UserPurchaseRepository
	processedCommandsRepo uow.ProcessedCommandRepository
	outboxRepo            uow.OutboxRepository
}

func (m *mockUnitOfWork) Tracks() uow.TrackRepository           { return m.tracksRepo }
func (m *mockUnitOfWork) Outbox() uow.OutboxRepository          { return m.outboxRepo }
func (m *mockUnitOfWork) Artists() uow.ArtistRepository         { return nil }
func (m *mockUnitOfWork) Albums() uow.AlbumRepository           { return nil }
func (m *mockUnitOfWork) Purchases() uow.UserPurchaseRepository { return m.purchasesRepo }
func (m *mockUnitOfWork) ProcessedCommands() uow.ProcessedCommandRepository {
	return m.processedCommandsRepo
}

type mockPurchaseRepo struct {
	existsByOrderFn func(ctx context.Context, orderID uuid.UUID) (bool, error)
	createBatchFn   func(ctx context.Context, purchases []*domain.UserPurchase) error
	revokeByOrderFn func(ctx context.Context, orderID uuid.UUID) error
	hasTrackFn      func(ctx context.Context, userID, trackID uuid.UUID) (bool, error)
	hasAlbumFn      func(ctx context.Context, userID, albumID uuid.UUID) (bool, error)
	listByUserFn    func(ctx context.Context, userID uuid.UUID, itemType *domain.ItemType, limit, offset int) ([]*domain.UserPurchase, error)
}

func (m *mockPurchaseRepo) Create(ctx context.Context, purchase *domain.UserPurchase) error {
	return nil
}

func (m *mockPurchaseRepo) CreateBatch(ctx context.Context, purchases []*domain.UserPurchase) error {
	if m.createBatchFn != nil {
		return m.createBatchFn(ctx, purchases)
	}
	return nil
}

func (m *mockPurchaseRepo) HasPurchasedTrack(ctx context.Context, userID, trackID uuid.UUID) (bool, error) {
	if m.hasTrackFn != nil {
		return m.hasTrackFn(ctx, userID, trackID)
	}
	return false, nil
}

func (m *mockPurchaseRepo) HasPurchasedAlbum(ctx context.Context, userID, albumID uuid.UUID) (bool, error) {
	if m.hasAlbumFn != nil {
		return m.hasAlbumFn(ctx, userID, albumID)
	}
	return false, nil
}

func (m *mockPurchaseRepo) GetPurchasedTrackIDsIn(ctx context.Context, userID uuid.UUID, trackIDs []uuid.UUID) (map[uuid.UUID]bool, error) {
	return nil, nil
}

func (m *mockPurchaseRepo) ListByUserID(ctx context.Context, userID uuid.UUID, itemType *domain.ItemType, limit, offset int) ([]*domain.UserPurchase, error) {
	if m.listByUserFn != nil {
		return m.listByUserFn(ctx, userID, itemType, limit, offset)
	}
	return nil, nil
}

func (m *mockPurchaseRepo) ExistsByOrderID(ctx context.Context, orderID uuid.UUID) (bool, error) {
	if m.existsByOrderFn != nil {
		return m.existsByOrderFn(ctx, orderID)
	}
	return false, nil
}

func (m *mockPurchaseRepo) RevokeByOrderID(ctx context.Context, orderID uuid.UUID) error {
	if m.revokeByOrderFn != nil {
		return m.revokeByOrderFn(ctx, orderID)
	}
	return nil
}

type mockTrackRepo struct {
	getByIDFn      func(ctx context.Context, id uuid.UUID) (*domain.Track, error)
	getByAlbumIDFn func(ctx context.Context, albumID uuid.UUID) ([]*domain.Track, error)
}

func (m *mockTrackRepo) Create(ctx context.Context, track *domain.Track) error { return nil }
func (m *mockTrackRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Track, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, id)
	}
	return nil, nil
}
func (m *mockTrackRepo) GetByArtistID(ctx context.Context, artistID uuid.UUID, limit, offset int) ([]*domain.Track, error) {
	return nil, nil
}
func (m *mockTrackRepo) GetByAlbumID(ctx context.Context, albumID uuid.UUID) ([]*domain.Track, error) {
	if m.getByAlbumIDFn != nil {
		return m.getByAlbumIDFn(ctx, albumID)
	}
	return nil, nil
}
func (m *mockTrackRepo) GetByIDs(ctx context.Context, ids []uuid.UUID) ([]*domain.Track, error) {
	return nil, nil
}
func (m *mockTrackRepo) Update(ctx context.Context, track *domain.Track) error { return nil }
func (m *mockTrackRepo) UpdateStatus(ctx context.Context, id uuid.UUID, status domain.TrackStatus, fileKey, previewKey string, duration time.Duration) error {
	return nil
}
func (m *mockTrackRepo) Delete(ctx context.Context, id uuid.UUID) error { return nil }

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

type mockOutboxRepo struct {
	recordFn func(ctx context.Context, event *domain.Event) error
}

func (m *mockOutboxRepo) Record(ctx context.Context, event *domain.Event) error {
	if m.recordFn != nil {
		return m.recordFn(ctx, event)
	}
	return nil
}

func newDiscardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func TestProcessGrantEntitlements_SuccessWithFanOutAndDeduplication(t *testing.T) {
	buyerID := uuid.New()
	orderID := uuid.New()
	sagaID := uuid.New()
	cmdID := uuid.New()
	trackID1 := uuid.New()
	trackID2 := uuid.New()
	albumID := uuid.New()

	var capturedBatch []*domain.UserPurchase
	var capturedEvent *domain.Event
	var markedCommandID, markedSagaID uuid.UUID
	var markedCommandName string

	purchasesRepo := &mockPurchaseRepo{
		createBatchFn: func(ctx context.Context, purchases []*domain.UserPurchase) error {
			capturedBatch = purchases
			return nil
		},
	}

	tracksRepo := &mockTrackRepo{
		getByAlbumIDFn: func(ctx context.Context, aid uuid.UUID) ([]*domain.Track, error) {
			artistID := uuid.New()
			t1 := domain.RestoreTrack(trackID1, artistID, &aid, "T1", 200*time.Second, "", "", "", "", 100, domain.CurrencyUSD, domain.TrackStatusReady, time.Now(), time.Now())
			t2 := domain.RestoreTrack(trackID2, artistID, &aid, "T2", 200*time.Second, "", "", "", "", 100, domain.CurrencyUSD, domain.TrackStatusReady, time.Now(), time.Now())
			return []*domain.Track{t1, t2}, nil
		},
	}

	inboxRepo := &mockInboxRepo{
		hasBeenProcessedFn: func(ctx context.Context, commandID uuid.UUID) (bool, error) {
			return false, nil
		},
		markProcessedFn: func(ctx context.Context, commandID, sID uuid.UUID, commandName string) error {
			markedCommandID = commandID
			markedSagaID = sID
			markedCommandName = commandName
			return nil
		},
	}

	outboxRepo := &mockOutboxRepo{
		recordFn: func(ctx context.Context, event *domain.Event) error {
			capturedEvent = event
			return nil
		},
	}

	uowWork := &mockUnitOfWork{
		tracksRepo:            tracksRepo,
		purchasesRepo:         purchasesRepo,
		processedCommandsRepo: inboxRepo,
		outboxRepo:            outboxRepo,
	}
	uowMgr := &mockUoWManager{work: uowWork}

	svc := user_purchase.NewService(uowMgr, purchasesRepo, tracksRepo, nil, newDiscardLogger())

	input := user_purchase.GrantEntitlementsInput{
		CommandID:   cmdID,
		SagaID:      sagaID,
		OrderID:     orderID,
		BuyerUserID: buyerID,
		Items: []user_purchase.PaidItemInput{
			{ItemID: trackID1, ItemType: itemtype.Track},
			{ItemID: albumID, ItemType: itemtype.Album},
		},
	}

	if err := svc.ProcessGrantEntitlements(context.Background(), input); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(capturedBatch) != 3 {
		t.Fatalf("expected 3 items in batch, got %d", len(capturedBatch))
	}

	var hasTrack1, hasTrack2, hasAlbum bool
	for _, p := range capturedBatch {
		if p.IsTrack() && p.ItemID() == trackID1 {
			hasTrack1 = true
		}
		if p.IsTrack() && p.ItemID() == trackID2 {
			hasTrack2 = true
		}
		if p.IsAlbum() && p.ItemID() == albumID {
			hasAlbum = true
		}
	}

	if !hasTrack1 || !hasTrack2 || !hasAlbum {
		t.Errorf("missing expected items in batch: track1=%v, track2=%v, album=%v", hasTrack1, hasTrack2, hasAlbum)
	}

	if capturedEvent == nil {
		t.Fatal("expected outbox event to be recorded")
	}
	if capturedEvent.EventType != domain.EventTypeSagaReplyCatalogGranted {
		t.Errorf("expected event type %s, got %s", domain.EventTypeSagaReplyCatalogGranted, capturedEvent.EventType)
	}

	var replyPayload user_purchase.CatalogReplyPayload
	if err := json.Unmarshal(capturedEvent.Payload, &replyPayload); err != nil {
		t.Fatalf("failed to unmarshal reply payload: %v", err)
	}
	if !replyPayload.Success {
		t.Errorf("expected success to be true")
	}
	if replyPayload.OrderID != orderID {
		t.Errorf("expected order id %s, got %s", orderID, replyPayload.OrderID)
	}
	if replyPayload.SagaID != sagaID {
		t.Errorf("expected saga id %s, got %s", sagaID, replyPayload.SagaID)
	}

	if markedCommandID != cmdID || markedSagaID != sagaID || markedCommandName != "catalog.grant_entitlements" {
		t.Errorf("unexpected inbox mark: cmd=%s, saga=%s, name=%s", markedCommandID, markedSagaID, markedCommandName)
	}
}

func TestProcessGrantEntitlements_IdempotencyFastPath(t *testing.T) {
	orderID := uuid.New()
	buyerID := uuid.New()
	cmdID := uuid.New()
	sagaID := uuid.New()

	batchCalled := false
	outboxCalled := false

	inboxRepo := &mockInboxRepo{
		hasBeenProcessedFn: func(ctx context.Context, commandID uuid.UUID) (bool, error) {
			return true, nil
		},
	}
	purchasesRepo := &mockPurchaseRepo{
		createBatchFn: func(ctx context.Context, purchases []*domain.UserPurchase) error {
			batchCalled = true
			return nil
		},
	}
	outboxRepo := &mockOutboxRepo{
		recordFn: func(ctx context.Context, event *domain.Event) error {
			outboxCalled = true
			return nil
		},
	}

	uowWork := &mockUnitOfWork{
		processedCommandsRepo: inboxRepo,
		purchasesRepo:         purchasesRepo,
		outboxRepo:            outboxRepo,
	}
	uowMgr := &mockUoWManager{work: uowWork}

	svc := user_purchase.NewService(uowMgr, purchasesRepo, nil, nil, newDiscardLogger())

	input := user_purchase.GrantEntitlementsInput{
		CommandID:   cmdID,
		SagaID:      sagaID,
		OrderID:     orderID,
		BuyerUserID: buyerID,
		Items: []user_purchase.PaidItemInput{
			{ItemID: uuid.New(), ItemType: itemtype.Track},
		},
	}

	if err := svc.ProcessGrantEntitlements(context.Background(), input); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if batchCalled {
		t.Error("expected batch create not to be called when command already processed")
	}
	if outboxCalled {
		t.Error("expected outbox record not to be called when command already processed")
	}
}

func TestProcessGrantEntitlements_ValidationError(t *testing.T) {
	svc := user_purchase.NewService(nil, nil, nil, nil, newDiscardLogger())

	err := svc.ProcessGrantEntitlements(context.Background(), user_purchase.GrantEntitlementsInput{})
	if err == nil {
		t.Fatal("expected error on empty input")
	}
}

func TestRevokePurchases_Success(t *testing.T) {
	cmdID := uuid.New()
	sagaID := uuid.New()
	orderID := uuid.New()

	var revokedOrderID uuid.UUID
	var capturedEvent *domain.Event
	var markedCommandID, markedSagaID uuid.UUID
	var markedCommandName string

	purchasesRepo := &mockPurchaseRepo{
		revokeByOrderFn: func(ctx context.Context, oid uuid.UUID) error {
			revokedOrderID = oid
			return nil
		},
	}

	inboxRepo := &mockInboxRepo{
		hasBeenProcessedFn: func(ctx context.Context, commandID uuid.UUID) (bool, error) {
			return false, nil
		},
		markProcessedFn: func(ctx context.Context, commandID, sID uuid.UUID, commandName string) error {
			markedCommandID = commandID
			markedSagaID = sID
			markedCommandName = commandName
			return nil
		},
	}

	outboxRepo := &mockOutboxRepo{
		recordFn: func(ctx context.Context, event *domain.Event) error {
			capturedEvent = event
			return nil
		},
	}

	uowWork := &mockUnitOfWork{
		purchasesRepo:         purchasesRepo,
		processedCommandsRepo: inboxRepo,
		outboxRepo:            outboxRepo,
	}
	uowMgr := &mockUoWManager{work: uowWork}

	svc := user_purchase.NewService(uowMgr, purchasesRepo, nil, nil, newDiscardLogger())

	input := user_purchase.RevokePurchasesInput{
		CommandID: cmdID,
		SagaID:    sagaID,
		OrderID:   orderID,
		Reason:    "payment refund",
	}

	if err := svc.RevokePurchases(context.Background(), input); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if revokedOrderID != orderID {
		t.Errorf("expected revoked order id %s, got %s", orderID, revokedOrderID)
	}

	if capturedEvent == nil {
		t.Fatal("expected outbox event to be recorded")
	}
	if capturedEvent.EventType != domain.EventTypeSagaReplyCatalogRevoked {
		t.Errorf("expected event type %s, got %s", domain.EventTypeSagaReplyCatalogRevoked, capturedEvent.EventType)
	}

	if markedCommandID != cmdID || markedSagaID != sagaID || markedCommandName != "catalog.revoke_entitlements" {
		t.Errorf("unexpected inbox mark: cmd=%s, saga=%s, name=%s", markedCommandID, markedSagaID, markedCommandName)
	}
}

func TestRevokePurchases_IdempotencyFastPath(t *testing.T) {
	cmdID := uuid.New()
	sagaID := uuid.New()
	orderID := uuid.New()

	revokeCalled := false
	outboxCalled := false

	inboxRepo := &mockInboxRepo{
		hasBeenProcessedFn: func(ctx context.Context, commandID uuid.UUID) (bool, error) {
			return true, nil
		},
	}
	purchasesRepo := &mockPurchaseRepo{
		revokeByOrderFn: func(ctx context.Context, oid uuid.UUID) error {
			revokeCalled = true
			return nil
		},
	}
	outboxRepo := &mockOutboxRepo{
		recordFn: func(ctx context.Context, event *domain.Event) error {
			outboxCalled = true
			return nil
		},
	}

	uowWork := &mockUnitOfWork{
		purchasesRepo:         purchasesRepo,
		processedCommandsRepo: inboxRepo,
		outboxRepo:            outboxRepo,
	}
	uowMgr := &mockUoWManager{work: uowWork}

	svc := user_purchase.NewService(uowMgr, purchasesRepo, nil, nil, newDiscardLogger())

	input := user_purchase.RevokePurchasesInput{
		CommandID: cmdID,
		SagaID:    sagaID,
		OrderID:   orderID,
	}

	if err := svc.RevokePurchases(context.Background(), input); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if revokeCalled {
		t.Error("expected revoke not to be called when command already processed")
	}
	if outboxCalled {
		t.Error("expected outbox record not to be called when command already processed")
	}
}

func TestRevokePurchases_ValidationError(t *testing.T) {
	svc := user_purchase.NewService(nil, nil, nil, nil, newDiscardLogger())

	err := svc.RevokePurchases(context.Background(), user_purchase.RevokePurchasesInput{})
	if err == nil {
		t.Fatal("expected error on empty input")
	}
}

func TestHasTrackAccess(t *testing.T) {
	trackID := uuid.New()
	userID := uuid.New()
	artistID := uuid.New()

	tests := []struct {
		name       string
		userID     *uuid.UUID
		trackPrice int64
		hasBought  bool
		wantAccess bool
	}{
		{
			name:       "free track accessible to guest",
			userID:     nil,
			trackPrice: 0,
			wantAccess: true,
		},
		{
			name:       "free track accessible to authenticated user",
			userID:     &userID,
			trackPrice: 0,
			wantAccess: true,
		},
		{
			name:       "paid track inaccessible to guest",
			userID:     nil,
			trackPrice: 100,
			wantAccess: false,
		},
		{
			name:       "paid track purchased by user",
			userID:     &userID,
			trackPrice: 100,
			hasBought:  true,
			wantAccess: true,
		},
		{
			name:       "paid track not purchased by user",
			userID:     &userID,
			trackPrice: 100,
			hasBought:  false,
			wantAccess: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tracksRepo := &mockTrackRepo{
				getByIDFn: func(ctx context.Context, id uuid.UUID) (*domain.Track, error) {
					return domain.RestoreTrack(trackID, artistID, nil, "T", 180*time.Second, "", "", "", "", tt.trackPrice, domain.CurrencyUSD, domain.TrackStatusReady, time.Now(), time.Now()), nil
				},
			}
			purchasesRepo := &mockPurchaseRepo{
				hasTrackFn: func(ctx context.Context, uid, tid uuid.UUID) (bool, error) {
					return tt.hasBought, nil
				},
			}

			svc := user_purchase.NewService(nil, purchasesRepo, tracksRepo, nil, newDiscardLogger())

			got, err := svc.HasTrackAccess(context.Background(), tt.userID, trackID)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.wantAccess {
				t.Errorf("HasTrackAccess() = %v, want %v", got, tt.wantAccess)
			}
		})
	}
}

func TestListUserPurchases(t *testing.T) {
	userID := uuid.New()
	orderID := uuid.New()
	trackID := uuid.New()
	p, err := domain.NewTrackPurchase(userID, orderID, trackID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	purchasesRepo := &mockPurchaseRepo{
		listByUserFn: func(ctx context.Context, uid uuid.UUID, itemType *domain.ItemType, limit, offset int) ([]*domain.UserPurchase, error) {
			return []*domain.UserPurchase{p}, nil
		},
	}

	svc := user_purchase.NewService(nil, purchasesRepo, nil, nil, newDiscardLogger())

	out, err := svc.ListUserPurchases(context.Background(), user_purchase.ListUserPurchasesInput{
		UserID: userID,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(out.Purchases) != 1 {
		t.Fatalf("expected 1 purchase, got %d", len(out.Purchases))
	}
	if out.Purchases[0].Status != domain.PurchaseStatusActive {
		t.Errorf("expected status %s, got %s", domain.PurchaseStatusActive, out.Purchases[0].Status)
	}
}
