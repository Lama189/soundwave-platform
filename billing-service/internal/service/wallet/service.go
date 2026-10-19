package wallet

import (
	"context"
	"errors"
	"fmt"

	"github.com/Lama189/soundwave-platform/billing-service/internal/domain"
	"github.com/Lama189/soundwave-platform/billing-service/internal/service/fx"
	"github.com/Lama189/soundwave-platform/billing-service/internal/uow"
	"github.com/google/uuid"
)

type WalletService struct {
	uow       UnitOfWorkManager
	converter CurrencyConverter
	wallets   WalletRepository
	ledgers   LedgerRepository
}

func NewWalletService(uow UnitOfWorkManager, converter CurrencyConverter, wallets WalletRepository, ledgers LedgerRepository) (*WalletService, error) {
	if converter == nil {
		return nil, errors.New("currency converter is not configured")
	}

	return &WalletService{
		uow:       uow,
		converter: converter,
		wallets:   wallets,
		ledgers:   ledgers,
	}, nil
}

func (s *WalletService) GetOrCreateWallet(ctx context.Context, artistID uuid.UUID) (*WalletOutputDTO, error) {
	if artistID == uuid.Nil {
		return nil, fmt.Errorf("%w: artist id is required", domain.ErrInvalidInput)
	}

	existing, err := s.wallets.GetByArtistID(ctx, artistID)
	if err == nil && existing != nil {
		return toWalletOutputDTO(existing, nil, nil), nil
	}

	if !errors.Is(err, domain.ErrNotFound) {
		return nil, fmt.Errorf("get wallet by artist id %s: %w", artistID, err)
	}

	new, err := domain.NewWallet(artistID)
	if err != nil {
		return nil, fmt.Errorf("init new wallet aggregate: %w", err)
	}

	if err := s.wallets.Create(ctx, new); err != nil {
		if errors.Is(err, domain.ErrConflict) {
			concurrentWallet, getErr := s.wallets.GetByArtistID(ctx, artistID)
			if getErr != nil {
				return nil, fmt.Errorf("fetch concurrently created wallet for artist %s: %w", artistID, getErr)
			}
			return toWalletOutputDTO(concurrentWallet, nil, nil), nil
		}
		return nil, fmt.Errorf("create wallet in repository: %w", err)
	}

	return toWalletOutputDTO(new, nil, nil), nil
}

func (s *WalletService) GetWallet(ctx context.Context, input GetWalletDTO) (*WalletOutputDTO, error) {
	if input.ArtistID == uuid.Nil {
		return nil, fmt.Errorf("%w: artist id is required", domain.ErrInvalidInput)
	}

	wallet, err := s.wallets.GetByArtistID(ctx, input.ArtistID)
	if err != nil {
		return nil, fmt.Errorf("get wallet for artist %s: %w", input.ArtistID, err)
	}

	if input.TargetCurrency == nil || *input.TargetCurrency == wallet.Currency {
		return toWalletOutputDTO(wallet, nil, nil), nil
	}

	convertedBal, err := s.converter.Convert(ctx, fx.ConvertCurrency{
		Source:         wallet.Balance,
		TargetCurrency: *input.TargetCurrency,
	})
	if err != nil {
		return nil, fmt.Errorf("convert balance to %s: %w", *input.TargetCurrency, err)
	}

	convertedLocked, err := s.converter.Convert(ctx, fx.ConvertCurrency{
		Source:         wallet.LockedBalance,
		TargetCurrency: *input.TargetCurrency,
	})
	if err != nil {
		return nil, fmt.Errorf("convert locked balance to %s: %w", *input.TargetCurrency, err)
	}

	return toWalletOutputDTO(wallet, &convertedBal.Converted, &convertedLocked.Converted), nil
}

func (s *WalletService) CreditBalance(ctx context.Context, input CreditBalanceDTO) (*CreditBalanceOutputDTO, error) {
	if input.ArtistID == uuid.Nil {
		return nil, fmt.Errorf("%w: artist id is requered", domain.ErrInvalidInput)
	}
	if err := input.Amount.Validate(); err != nil {
		return nil, fmt.Errorf("invalid credit amount: %w", err)
	}
	if !input.Amount.IsPositive() {
		return nil, fmt.Errorf("%w: amount must be positive", domain.ErrInvalidInput)
	}
	if input.ReferenceID == uuid.Nil {
		return nil, fmt.Errorf("%w: reference id is required", domain.ErrInvalidInput)
	}
	if !input.ReferenceType.IsValid() {
		return nil, fmt.Errorf("%w: invalid reference type %q", domain.ErrInvalidInput, input.ReferenceType)
	}

	var output CreditBalanceOutputDTO
	err := s.uow.Do(ctx, func(work uow.UnitOfWork) error {
		wallet, err := work.Wallets().GetByArtistIDForUpdate(ctx, input.ArtistID)
		if err != nil {
			return fmt.Errorf("lock wallet for update: %w", err)
		}

		if err := wallet.Credit(input.Amount); err != nil {
			return fmt.Errorf("Wallet credit domain error: %w", err)
		}

		if err := work.Wallets().Update(ctx, wallet); err != nil {
			return fmt.Errorf("update wallet balance: %w", err)
		}

		entry, err := domain.NewLedgerEntry(
			wallet.ID,
			domain.LedgerOpTrackSale,
			input.Amount,
			wallet.Balance,
			input.ReferenceType,
			input.ReferenceID,
			input.Description,
		)
		if err != nil {
			return fmt.Errorf("create ledger entry aggregate: %w", err)
		}

		if err := work.Ledger().Append(ctx, entry); err != nil {
			return fmt.Errorf("append ledger entry: %w", err)
		}

		output = CreditBalanceOutputDTO{
			WalletID:      wallet.ID,
			NewBalance:    wallet.Balance,
			LedgerEntryID: entry.ID,
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("credit balance transaction: %w", err)
	}

	return &output, nil
}

func (s *WalletService) GetStatement(ctx context.Context, input GetStatementDTO) (*StatementOutputDTO, error) {
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

	wallet, err := s.wallets.GetByArtistID(ctx, input.ArtistID)
	if err != nil {
		return nil, fmt.Errorf("get wallet for artist %s: %w", input.ArtistID, err)
	}

	entries, err := s.ledgers.GetByWalletID(ctx, wallet.ID, input.Limit, input.Offset)
	if err != nil {
		return nil, fmt.Errorf("get ledger statement for wallet %s: %w", wallet.ID, err)
	}

	return &StatementOutputDTO{
		WalletID: wallet.ID,
		Entries:  toLedgerEntryOutputDTOList(entries),
	}, nil
}
