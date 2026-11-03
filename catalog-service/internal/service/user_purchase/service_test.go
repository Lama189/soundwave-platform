package user_purchase_test

import (
	"context"
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
	tracksRepo    uow.TrackRepository
	purchasesRepo uow.UserPurchaseRepository
}

func (m *mockUnitOfWork) Tracks() uow.TrackRepository           { return m.tracksRepo }
func (m *mockUnitOfWork) Outbox() uow.OutboxRepository          { return nil }
func (m *mockUnitOfWork) Artists() uow.ArtistRepository         { return nil }
func (m *mockUnitOfWork) Albums() uow.AlbumRepository           { return nil }
func (m *mockUnitOfWork) Purchases() uow.UserPurchaseRepository { return m.purchasesRepo }

type mockPurchaseRepo struct {
	existsByOrderFn func(ctx context.Context, orderID uuid.UUID) (bool, error)
	createBatchFn   func(ctx context.Context, purchases []*domain.UserPurchase) error
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

type mockAlbumRepo struct {
	getByIDFn func(ctx context.Context, id uuid.UUID) (*domain.Album, error)
}

func (m *mockAlbumRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Album, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, id)
	}
	return nil, nil
}

func newDiscardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func TestProcessPaidOrder_SuccessWithFanOutAndDeduplication(t *testing.T) {
	buyerID := uuid.New()
	orderID := uuid.New()
	trackID1 := uuid.New()
	trackID2 := uuid.New()
	albumID := uuid.New()

	var capturedBatch []*domain.UserPurchase

	purchasesRepo := &mockPurchaseRepo{
		existsByOrderFn: func(ctx context.Context, oid uuid.UUID) (bool, error) {
			return false, nil
		},
		createBatchFn: func(ctx context.Context, purchases []*domain.UserPurchase) error {
			capturedBatch = purchases
			return nil
		},
	}

	tracksRepo := &mockTrackRepo{
		getByAlbumIDFn: func(ctx context.Context, aid uuid.UUID) ([]*domain.Track, error) {
			artistID := uuid.New()
			// Album contains trackID1 (already in order) and trackID2 (new)
			t1 := domain.RestoreTrack(trackID1, artistID, &aid, "T1", 200*time.Second, "", "", "", "", 100, domain.CurrencyUSD, domain.TrackStatusReady, time.Now(), time.Now())
			t2 := domain.RestoreTrack(trackID2, artistID, &aid, "T2", 200*time.Second, "", "", "", "", 100, domain.CurrencyUSD, domain.TrackStatusReady, time.Now(), time.Now())
			return []*domain.Track{t1, t2}, nil
		},
	}

	uowWork := &mockUnitOfWork{
		tracksRepo:    tracksRepo,
		purchasesRepo: purchasesRepo,
	}
	uowMgr := &mockUoWManager{work: uowWork}

	svc := user_purchase.NewService(uowMgr, purchasesRepo, tracksRepo, nil, newDiscardLogger())

	input := user_purchase.ProcessPaidOrderInput{
		OrderID:     orderID,
		BuyerUserID: buyerID,
		Items: []user_purchase.PaidItemInput{
			{ItemID: trackID1, ItemType: itemtype.Track},
			{ItemID: albumID, ItemType: itemtype.Album},
		},
	}

	if err := svc.ProcessPaidOrder(context.Background(), input); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Expect 3 items in batch:
	// 1. Explicit trackID1
	// 2. Explicit albumID
	// 3. Child trackID2 from album (trackID1 is deduplicated!)
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
}

func TestProcessPaidOrder_IdempotencyFastPath(t *testing.T) {
	orderID := uuid.New()
	buyerID := uuid.New()

	batchCalled := false
	purchasesRepo := &mockPurchaseRepo{
		existsByOrderFn: func(ctx context.Context, oid uuid.UUID) (bool, error) {
			return true, nil // Already processed
		},
		createBatchFn: func(ctx context.Context, purchases []*domain.UserPurchase) error {
			batchCalled = true
			return nil
		},
	}

	uowMgr := &mockUoWManager{}
	svc := user_purchase.NewService(uowMgr, purchasesRepo, nil, nil, newDiscardLogger())

	input := user_purchase.ProcessPaidOrderInput{
		OrderID:     orderID,
		BuyerUserID: buyerID,
		Items: []user_purchase.PaidItemInput{
			{ItemID: uuid.New(), ItemType: itemtype.Track},
		},
	}

	if err := svc.ProcessPaidOrder(context.Background(), input); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if batchCalled {
		t.Error("expected batch create not to be called when order already exists")
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
