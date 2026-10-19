package payout

import (
	"context"
	"fmt"
	"strings"

	"github.com/Lama189/soundwave-platform/billing-service/internal/domain"
	"github.com/Lama189/soundwave-platform/billing-service/internal/uow"
	"github.com/google/uuid"
)

type PayoutService struct {
	uow     UnitOfWorkManager
	payouts PayoutRepository
	wallets WalletRepository
}

func NewPayoutService(uow UnitOfWorkManager, payouts PayoutRepository, wallets WalletRepository) *PayoutService {
	return &PayoutService{
		uow:     uow,
		payouts: payouts,
		wallets: wallets,
	}
}

func (s *PayoutService) RequestPayout(ctx context.Context, input RequestPayoutDTO) (*PayoutOutputDTO, error) {
	if input.ArtistID == uuid.Nil {
		return nil, fmt.Errorf("%w: artist id is required", domain.ErrInvalidInput)
	}
	if err := input.Amount.Validate(); err != nil {
		return nil, fmt.Errorf("invalid payout amount: %w", err)
	}
	if !input.Amount.IsPositive() {
		return nil, fmt.Errorf("%w: payout amount must be positive", domain.ErrInvalidInput)
	}

	destCard := strings.TrimSpace(input.DestinationCard)
	if destCard == "" {
		return nil, fmt.Errorf("%w: destination card cannot be empty", domain.ErrInvalidInput)
	}

	var createdPayout *domain.Payout

	err := s.uow.Do(ctx, func(work uow.UnitOfWork) error {
		wallet, err := work.Wallets().GetByArtistIDForUpdate(ctx, input.ArtistID)
		if err != nil {
			return fmt.Errorf("lock wallet for payout: %w", err)
		}

		if err := wallet.LockForPayout(input.Amount); err != nil {
			return fmt.Errorf("wallet lock funds error: %w", err)
		}

		payout, err := domain.NewPayout(input.ArtistID, wallet.ID, input.Amount, destCard)
		if err != nil {
			return fmt.Errorf("create payout aggregate: %w", err)
		}

		debitAmount := domain.Money{
			Amount:   -input.Amount.Amount,
			Currency: input.Amount.Currency,
		}

		ledgerEntry, err := domain.NewLedgerEntry(
			wallet.ID,
			domain.LedgerOpPayoutLock,
			debitAmount,
			wallet.Balance,
			domain.ReferenceTypePayout,
			payout.ID,
			fmt.Sprintf("Funds locked for payout %s", payout.ID),
		)
		if err != nil {
			return fmt.Errorf("create payout lock ledger entry: %w", err)
		}

		event, err := domain.NewEvent(
			domain.AggregateTypePayout,
			payout.ID,
			domain.EventTypePayoutRequested,
			payout,
		)
		if err != nil {
			return fmt.Errorf("create payout outbox event: %w", err)
		}

		if err := work.Wallets().Update(ctx, wallet); err != nil {
			return fmt.Errorf("update wallet: %w", err)
		}
		if err := work.Payouts().Create(ctx, payout); err != nil {
			return fmt.Errorf("save payout: %w", err)
		}
		if err := work.Ledger().Append(ctx, ledgerEntry); err != nil {
			return fmt.Errorf("append ledger entry: %w", err)
		}
		if err := work.Outbox().Record(ctx, event); err != nil {
			return fmt.Errorf("record outbox event: %w", err)
		}

		createdPayout = payout
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("request payout transaction: %w", err)
	}

	return toPayoutOutputDTO(createdPayout), nil
}

func (s *PayoutService) ProcessPayout(ctx context.Context, payoutID uuid.UUID) (*PayoutOutputDTO, error) {
	if payoutID == uuid.Nil {
		return nil, fmt.Errorf("%w: payout id is required", domain.ErrInvalidInput)
	}

	var updatedPayout *domain.Payout
	err := s.uow.Do(ctx, func(work uow.UnitOfWork) error {
		payout, err := work.Payouts().GetByID(ctx, payoutID)
		if err != nil {
			return fmt.Errorf("get payout %s: %w", payoutID, err)
		}

		if payout.Status == domain.PayoutStatusProcessing {
			updatedPayout = payout
			return nil
		}

		if err := payout.MarkProcessing(); err != nil {
			return err
		}

		if err := work.Payouts().Update(ctx, payout); err != nil {
			return fmt.Errorf("update payout status: %w", err)
		}

		updatedPayout = payout
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("process payout: %w", err)
	}

	return toPayoutOutputDTO(updatedPayout), nil
}

func (s *PayoutService) CompletePayout(ctx context.Context, input CompletePayoutDTO) (*PayoutOutputDTO, error) {
	if input.PayoutID == uuid.Nil {
		return nil, fmt.Errorf("%w: payout id is required", domain.ErrInvalidInput)
	}

	var completedPayout *domain.Payout
	err := s.uow.Do(ctx, func(work uow.UnitOfWork) error {
		payout, err := work.Payouts().GetByID(ctx, input.PayoutID)
		if err != nil {
			return fmt.Errorf("get payout %s: %w", input.PayoutID, err)
		}

		if payout.Status == domain.PayoutStatusCompleted {
			completedPayout = payout
			return nil
		}

		wallet, err := work.Wallets().GetByArtistIDForUpdate(ctx, payout.ArtistID)
		if err != nil {
			return fmt.Errorf("lock wallet for payout confirmation: %w", err)
		}

		if err := wallet.ConfirmPayout(payout.Amount); err != nil {
			return fmt.Errorf("confirm payout in wallet: %w", err)
		}

		if err := payout.Complete(); err != nil {
			return fmt.Errorf("complete payout: %w", err)
		}

		debitAmount := domain.Money{
			Amount:   -payout.Amount.Amount,
			Currency: payout.Amount.Currency,
		}
		ledgerEntry, err := domain.NewLedgerEntry(
			wallet.ID,
			domain.LedgerOpPayoutSuccess,
			debitAmount,
			wallet.Balance,
			domain.ReferenceTypePayout,
			payout.ID,
			fmt.Sprintf("Payout %s successfully completed", payout.ID),
		)
		if err != nil {
			return fmt.Errorf("create payout success ledger entry: %w", err)
		}

		event, err := domain.NewEvent(
			domain.AggregateTypePayout,
			payout.ID,
			domain.EventTypePayoutCompleted,
			payout,
		)
		if err != nil {
			return fmt.Errorf("create payout completed event: %w", err)
		}

		if err := work.Wallets().Update(ctx, wallet); err != nil {
			return fmt.Errorf("update wallet: %w", err)
		}
		if err := work.Payouts().Update(ctx, payout); err != nil {
			return fmt.Errorf("update payout: %w", err)
		}
		if err := work.Ledger().Append(ctx, ledgerEntry); err != nil {
			return fmt.Errorf("append ledger entry: %w", err)
		}
		if err := work.Outbox().Record(ctx, event); err != nil {
			return fmt.Errorf("record outbox event: %w", err)
		}

		completedPayout = payout
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("complete payout transaction: %w", err)
	}

	return toPayoutOutputDTO(completedPayout), nil
}

func (s *PayoutService) FailPayout(ctx context.Context, input FailPayoutDTO) (*PayoutOutputDTO, error) {
	if input.PayoutID == uuid.Nil {
		return nil, fmt.Errorf("%w: payout id is required", domain.ErrInvalidInput)
	}
	reason := strings.TrimSpace(input.Reason)
	if reason == "" {
		return nil, fmt.Errorf("%w: failure reason cannot be empty", domain.ErrInvalidInput)
	}

	var failedPayout *domain.Payout

	err := s.uow.Do(ctx, func(work uow.UnitOfWork) error {
		payout, err := work.Payouts().GetByID(ctx, input.PayoutID)
		if err != nil {
			return fmt.Errorf("get payout %s: %w", input.PayoutID, err)
		}

		if payout.Status == domain.PayoutStatusFailed {
			failedPayout = payout
			return nil
		}

		wallet, err := work.Wallets().GetByArtistIDForUpdate(ctx, payout.ArtistID)
		if err != nil {
			return fmt.Errorf("lock wallet for payout unlock: %w", err)
		}

		if err := wallet.Unlock(payout.Amount); err != nil {
			return fmt.Errorf("unlock wallet funds: %w", err)
		}

		if err := payout.Fail(reason); err != nil {
			return fmt.Errorf("fail payout: %w", err)
		}

		ledgerEntry, err := domain.NewLedgerEntry(
			wallet.ID,
			domain.LedgerOpPayoutRefund,
			payout.Amount,
			wallet.Balance,
			domain.ReferenceTypePayout,
			payout.ID,
			fmt.Sprintf("Payout %s refunded: %s", payout.ID, reason),
		)
		if err != nil {
			return fmt.Errorf("create payout refund ledger entry: %w", err)
		}

		event, err := domain.NewEvent(
			domain.AggregateTypePayout,
			payout.ID,
			domain.EventTypePayoutFailed,
			payout,
		)
		if err != nil {
			return fmt.Errorf("create payout failed event: %w", err)
		}

		if err := work.Wallets().Update(ctx, wallet); err != nil {
			return fmt.Errorf("update wallet: %w", err)
		}
		if err := work.Payouts().Update(ctx, payout); err != nil {
			return fmt.Errorf("update payout: %w", err)
		}
		if err := work.Ledger().Append(ctx, ledgerEntry); err != nil {
			return fmt.Errorf("append ledger entry: %w", err)
		}
		if err := work.Outbox().Record(ctx, event); err != nil {
			return fmt.Errorf("record outbox event: %w", err)
		}

		failedPayout = payout
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("fail payout transaction: %w", err)
	}

	return toPayoutOutputDTO(failedPayout), nil
}

func (s *PayoutService) GetPayout(ctx context.Context, payoutID uuid.UUID) (*PayoutOutputDTO, error) {
	if payoutID == uuid.Nil {
		return nil, fmt.Errorf("%w: payout id is required", domain.ErrInvalidInput)
	}

	payout, err := s.payouts.GetByID(ctx, payoutID)
	if err != nil {
		return nil, fmt.Errorf("get payout %s: %w", payoutID, err)
	}

	return toPayoutOutputDTO(payout), nil
}

func (s *PayoutService) ListPayouts(ctx context.Context, input ListPayoutsDTO) (*PayoutListOutputDTO, error) {
	if input.ArtistID == uuid.Nil {
		return nil, fmt.Errorf("%w: artist id is required", domain.ErrInvalidInput)
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

	payouts, err := s.payouts.GetByArtistID(ctx, input.ArtistID, input.Limit, input.Offset)
	if err != nil {
		return nil, fmt.Errorf("get payouts for artist %s: %w", input.ArtistID, err)
	}

	filtered := make([]*domain.Payout, 0, len(payouts))
	for _, p := range payouts {
		if input.Status != nil && p.Status != *input.Status {
			continue
		}
		filtered = append(filtered, p)
	}

	return &PayoutListOutputDTO{
		Payouts: toPayoutOutputDTOList(filtered),
		Total:   len(filtered),
	}, nil
}
