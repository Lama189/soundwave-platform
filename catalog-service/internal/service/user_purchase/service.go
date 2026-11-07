package user_purchase

import (
	"context"
	"encoding/json"
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

func (s *Service) ProcessGrantEntitlements(ctx context.Context, input GrantEntitlementsInput) error {
	if err := input.Validate(); err != nil {
		return err
	}

	return s.uow.Do(ctx, func(work uow.UnitOfWork) error {
		alreadyProcessed, err := work.ProcessedCommands().HasBeenProcessed(ctx, input.CommandID)
		if err != nil {
			return fmt.Errorf("check inbox: %w", err)
		}
		if alreadyProcessed {
			s.logger.Info("command already processed, skipping",
				slog.String("command_id", input.CommandID.String()))
			return nil
		}

		var purchasesToCreate []*domain.UserPurchase
		seenTracks := make(map[uuid.UUID]struct{})

		for _, item := range input.Items {
			switch item.ItemType {
			case itemtype.Track:
				if _, seen := seenTracks[item.ItemID]; seen {
					continue
				}
				seenTracks[item.ItemID] = struct{}{}
				p, err := domain.NewTrackPurchase(input.BuyerUserID, input.OrderID, item.ItemID)
				if err != nil {
					return fmt.Errorf("init track purchase %s: %w", item.ItemID, err)
				}
				purchasesToCreate = append(purchasesToCreate, p)

			case itemtype.Album:
				albumPurchase, err := domain.NewAlbumPurchase(input.BuyerUserID, input.OrderID, item.ItemID)
				if err != nil {
					return fmt.Errorf("init album purchase %s: %w", item.ItemID, err)
				}
				purchasesToCreate = append(purchasesToCreate, albumPurchase)

				albumTracks, err := work.Tracks().GetByAlbumID(ctx, item.ItemID)
				if err != nil {
					return fmt.Errorf("fetch album %s tracks: %w", item.ItemID, err)
				}

				for _, track := range albumTracks {
					if _, seen := seenTracks[track.ID]; seen {
						continue
					}
					seenTracks[track.ID] = struct{}{}
					trackPurchase, err := domain.NewTrackPurchase(input.BuyerUserID, input.OrderID, track.ID)
					if err != nil {
						return fmt.Errorf("init album child track purchase: %w", err)
					}
					purchasesToCreate = append(purchasesToCreate, trackPurchase)
				}
			}
		}

		if err := work.Purchases().CreateBatch(ctx, purchasesToCreate); err != nil {
			return fmt.Errorf("save purchases batch: %w", err)
		}

		payloadBytes, err := json.Marshal(CatalogReplyPayload{
			CommandID: input.CommandID,
			SagaID:    input.SagaID,
			OrderID:   input.OrderID,
			Success:   true,
		})
		if err != nil {
			return fmt.Errorf("marshal reply payload: %w", err)
		}

		replyEvent, err := domain.NewEvent(
			uuid.New(),
			domain.AggregateTypeUserPurchase,
			input.OrderID,
			domain.EventTypeSagaReplyCatalogGranted,
			payloadBytes,
		)
		if err != nil {
			return fmt.Errorf("create outbox reply event: %w", err)
		}

		if err := work.Outbox().Record(ctx, replyEvent); err != nil {
			return fmt.Errorf("record outbox reply: %w", err)
		}

		if err := work.ProcessedCommands().MarkProcessed(ctx, input.CommandID, input.SagaID, "catalog.grant_entitlements"); err != nil {
			return fmt.Errorf("mark command in inbox: %w", err)
		}

		return nil
	})
}

func (s *Service) RevokePurchases(ctx context.Context, input RevokePurchasesInput) error {
	if err := input.Validate(); err != nil {
		return err
	}

	return s.uow.Do(ctx, func(work uow.UnitOfWork) error {
		alreadyProcessed, err := work.ProcessedCommands().HasBeenProcessed(ctx, input.CommandID)
		if err != nil {
			return fmt.Errorf("check inbox: %w", err)
		}
		if alreadyProcessed {
			return nil
		}

		if err := work.Purchases().RevokeByOrderID(ctx, input.OrderID); err != nil {
			return fmt.Errorf("revoke purchases: %w", err)
		}

		payloadBytes, err := json.Marshal(CatalogReplyPayload{
			CommandID: input.CommandID,
			SagaID:    input.SagaID,
			OrderID:   input.OrderID,
			Success:   true,
		})
		if err != nil {
			return fmt.Errorf("marshal revoke reply payload: %w", err)
		}

		replyEvent, err := domain.NewEvent(
			uuid.New(),
			domain.AggregateTypeUserPurchase,
			input.OrderID,
			domain.EventTypeSagaReplyCatalogRevoked,
			payloadBytes,
		)
		if err != nil {
			return fmt.Errorf("create outbox revoke event: %w", err)
		}

		if err := work.Outbox().Record(ctx, replyEvent); err != nil {
			return fmt.Errorf("record outbox reply: %w", err)
		}

		if err := work.ProcessedCommands().MarkProcessed(ctx, input.CommandID, input.SagaID, "catalog.revoke_entitlements"); err != nil {
			return fmt.Errorf("mark revoke command in inbox: %w", err)
		}

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
			Status:    p.Status,
			RevokedAt: p.RevokedAt,
			CreatedAt: p.CreatedAt,
		}
	}

	return &ListUserPurchasesOutput{Purchases: out}, nil
}
