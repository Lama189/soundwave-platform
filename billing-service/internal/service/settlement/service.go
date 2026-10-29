package settlement

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"github.com/Lama189/soundwave-platform/billing-service/internal/domain"
	"github.com/Lama189/soundwave-platform/billing-service/internal/uow"
)

type Service struct {
	uow                uow.Manager
	platformFeePercent int
}

func NewService(uow uow.Manager, platformFeePercent int) *Service {
	if platformFeePercent < 0 || platformFeePercent > 100 {
		platformFeePercent = 15
	}

	return &Service{
		uow:                uow,
		platformFeePercent: platformFeePercent,
	}
}

func (s *Service) ProcessPaidOrder(ctx context.Context, input ProcessPaidOrderInput) error {
	if input.OrderID == uuid.Nil {
		return fmt.Errorf("%w: order id is required", domain.ErrInvalidInput)
	}
	if len(input.Items) == 0 {
		return fmt.Errorf("%w: order items cannot be empty", domain.ErrInvalidInput)
	}

	return s.uow.Do(ctx, func(work uow.UnitOfWork) error {
		alreadyProcessed, err := work.ProcessedEvents().IsProcessed(ctx, input.OrderID)
		if err != nil {
			return fmt.Errorf("check order idempotency: %w", err)
		}
		if alreadyProcessed {
			return domain.ErrOrderAlreadyProcessed
		}

		for _, item := range input.Items {
			if item.ArtistID == uuid.Nil {
				continue
			}
			if !item.ItemType.IsValid() {
				return fmt.Errorf("%w: unsupported item type %q", domain.ErrInvalidInput, item.ItemType)
			}

			feeAmount := (item.Price * int64(s.platformFeePercent)) / 100
			artistShareAmount := item.Price - feeAmount

			artistShareMoney, err := domain.NewMoney(artistShareAmount, item.Currency)
			if err != nil {
				return fmt.Errorf("create artist share money: %w", err)
			}

			wallet, err := work.Wallets().GetByArtistIDForUpdate(ctx, item.ArtistID)
			if err != nil {
				if errors.Is(err, domain.ErrNotFound) {
					wallet, err = domain.NewWallet(item.ArtistID)
					if err != nil {
						return fmt.Errorf("init new wallet for artist %s: %w", item.ArtistID, err)
					}

					if err := work.Wallets().Create(ctx, wallet); err != nil {
						return fmt.Errorf("create wallet for artist %s: %w", item.ArtistID, err)
					}
				} else {
					return fmt.Errorf("get wallet for artist %s: %w", item.ArtistID, err)
				}
			}

			if err := wallet.Credit(artistShareMoney); err != nil {
				return fmt.Errorf("credit artist %s wallet: %w", item.ArtistID, err)
			}

			if err := work.Wallets().Update(ctx, wallet); err != nil {
				return fmt.Errorf("update wallet balance for artist %s: %w", item.ArtistID, err)
			}

			description := fmt.Sprintf(
				"Sale of %s %s (order: %s, gross price %d %s, fee: %d%%)",
				item.ItemType, item.ItemID, input.OrderID, item.Price, item.Currency, s.platformFeePercent,
			)

			opType := domain.LedgerOpTrackSale
			if item.ItemType == ItemTypeAlbum {
				opType = domain.LedgerOpAlbumSale
			}

			entry, err := domain.NewLedgerEntry(wallet.ID, opType, artistShareMoney, wallet.Balance, domain.ReferenceTypeOrder, input.OrderID, description)
			if err != nil {
				return fmt.Errorf("create ledger entry: %w", err)
			}

			if err := work.Ledger().Append(ctx, entry); err != nil {
				return fmt.Errorf("append ledger entry: %w", err)
			}
		}

		processedEvent, err := domain.NewProcessedEvent(input.OrderID, domain.InboxEventTypeOrderPaid)
		if err != nil {
			return fmt.Errorf("create processed event: %w", err)
		}

		if err := work.ProcessedEvents().MarkProcessed(ctx, processedEvent); err != nil {
			return fmt.Errorf("mark event as processed: %w", err)
		}

		return nil
	})
}
