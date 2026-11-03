package user_purchase

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/Lama189/soundwave-platform/catalog-service/internal/domain"
	"github.com/Lama189/soundwave-platform/catalog-service/internal/uow"
	"github.com/Lama189/soundwave-platform/pkg/itemtype"
	"github.com/google/uuid"
)

type Service struct {
	uow       UnitOfWorkManager
	purchases UserPurchaseRepository
	tracks    TrackRepository
	albums    AlbumRepository
	logger    *slog.Logger
}

func NewService(uow UnitOfWorkManager, purchases UserPurchaseRepository, tracks TrackRepository, albums AlbumRepository, logger *slog.Logger) *Service {
	return &Service{
		uow:       uow,
		purchases: purchases,
		tracks:    tracks,
		albums:    albums,
		logger:    logger,
	}
}

func (s *Service) ProcessPaidOrder(ctx context.Context, input ProcessPaidOrderInput) error {
	if err := input.Validate(); err != nil {
		return err
	}

	alreadyProcessed, err := s.purchases.ExistsByOrderID(ctx, input.OrderID)
	if err != nil {
		return fmt.Errorf("check order %s for idempotency: %w", input.OrderID, err)
	}
	if alreadyProcessed {
		s.logger.Info(
			"order already processed, skipping purchase creation",
			slog.String("order_id", input.OrderID.String()),
			slog.String("user_id", input.BuyerUserID.String()),
		)
		return nil
	}

	var explicitTrackIDs []uuid.UUID
	var explicitAlbumIDs []uuid.UUID

	for _, item := range input.Items {
		switch item.ItemType {
		case itemtype.Track:
			explicitTrackIDs = append(explicitTrackIDs, item.ItemID)
		case itemtype.Album:
			explicitAlbumIDs = append(explicitAlbumIDs, item.ItemID)
		}
	}

	return s.uow.Do(ctx, func(work uow.UnitOfWork) error {
		var purchasesToCreate []*domain.UserPurchase
		seenTracks := make(map[uuid.UUID]struct{})

		for _, trackID := range explicitTrackIDs {
			if _, seen := seenTracks[trackID]; seen {
				continue
			}
			seenTracks[trackID] = struct{}{}

			p, err := domain.NewTrackPurchase(input.BuyerUserID, input.OrderID, trackID)
			if err != nil {
				return fmt.Errorf("init track purchase %s: %w", trackID, err)
			}

			purchasesToCreate = append(purchasesToCreate, p)
		}

		for _, albumID := range explicitAlbumIDs {
			albumPurchase, err := domain.NewAlbumPurchase(input.BuyerUserID, input.OrderID, albumID)
			if err != nil {
				return fmt.Errorf("init album purchase %s: %w", albumID, err)
			}

			purchasesToCreate = append(purchasesToCreate, albumPurchase)

			albumTracks, err := work.Tracks().GetByAlbumID(ctx, albumID)
			if err != nil {
				return fmt.Errorf("fetch album %s tracks for purchase fan-out: %w", albumID, err)
			}

			for _, track := range albumTracks {
				if _, seen := seenTracks[track.ID]; seen {
					continue
				}

				seenTracks[track.ID] = struct{}{}

				trackPurchase, err := domain.NewTrackPurchase(input.BuyerUserID, input.OrderID, track.ID)
				if err != nil {
					return fmt.Errorf("init album child track purchase %s: %w", track.ID, err)
				}

				purchasesToCreate = append(purchasesToCreate, trackPurchase)
			}
		}

		if err := work.Purchases().CreateBatch(ctx, purchasesToCreate); err != nil {
			return fmt.Errorf("save batch purchases for order %s: %w", input.OrderID, err)
		}

		s.logger.Info(
			"order purchases recorded successfully",
			slog.String("order_id", input.OrderID.String()),
			slog.String("buyer_user_id", input.BuyerUserID.String()),
			slog.Int("total_entitlements", len(purchasesToCreate)),
		)

		return nil
	})
}

func (s *Service) HasTrackAccess(ctx context.Context, userID *uuid.UUID, trackID uuid.UUID) (bool, error) {
	if trackID == uuid.Nil {
		return false, fmt.Errorf("%w: track id is required", domain.ErrInvalidInput)
	}

	track, err := s.tracks.GetByID(ctx, trackID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) || errors.Is(err, domain.ErrTrackNotFound) {
			return false, domain.ErrTrackNotFound
		}

		return false, fmt.Errorf("find track for access check: %w", err)
	}

	if track.Price == 0 {
		return true, nil
	}

	if userID == nil || *userID == uuid.Nil {
		return false, nil
	}

	return s.purchases.HasPurchasedTrack(ctx, *userID, trackID)
}

func (s *Service) ListUserPurchases(ctx context.Context, input ListUserPurchasesInput) (*ListUserPurchasesOutput, error) {
	if input.UserID == uuid.Nil {
		return nil, fmt.Errorf("%w: user id is required", domain.ErrInvalidInput)
	}

	if input.Limit <= 0 {
		input.Limit = 20
	}
	if input.Limit > 100 {
		input.Limit = 100
	}
	if input.Offset < 0 {
		input.Offset = 0
	}

	purchases, err := s.purchases.ListByUserID(ctx, input.UserID, input.ItemType, input.Limit, input.Offset)
	if err != nil {
		return nil, fmt.Errorf("list user purchases: %w", err)
	}

	out := make([]*UserPurchaseOutput, len(purchases))
	for i, p := range purchases {
		out[i] = &UserPurchaseOutput{
			ID:        p.ID,
			UserID:    p.UserID,
			OrderID:   p.OrderID,
			ItemType:  p.ItemType,
			TrackID:   p.TrackID,
			AlbumID:   p.AlbumID,
			CreatedAt: p.CreatedAt,
		}
	}

	return &ListUserPurchasesOutput{Purchases: out}, nil
}
