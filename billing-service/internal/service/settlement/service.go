package settlement

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"github.com/Lama189/soundwave-platform/billing-service/internal/domain"
	"github.com/Lama189/soundwave-platform/billing-service/internal/uow"
	"github.com/Lama189/soundwave-platform/pkg/itemtype"
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

func (s *Service) ProcessSettleOrder(ctx context.Context, input SettleOrderInput) error {
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

		for _, item := range input.Items {
			if item.ArtistID == uuid.Nil {
				continue
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
			if item.ItemType == itemtype.Album {
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

		replyPayload := BillingReplyPayload{
			CommandID: input.CommandID,
			SagaID:    input.SagaID,
			OrderID:   input.OrderID,
			Success:   true,
		}

		replyEvent, err := domain.NewEvent(
			domain.AggregateTypeOrder,
			input.OrderID,
			domain.EventTypeSagaReplyBillingSettled,
			replyPayload,
		)
		if err != nil {
			return fmt.Errorf("create outbox reply event: %w", err)
		}

		if err := work.Outbox().Record(ctx, replyEvent); err != nil {
			return fmt.Errorf("record outbox reply: %w", err)
		}

		if err := work.ProcessedCommands().MarkProcessed(ctx, input.CommandID, input.SagaID, "billing.settle_order"); err != nil {
			return fmt.Errorf("mark command in inbox: %w", err)
		}

		return nil
	})
}
