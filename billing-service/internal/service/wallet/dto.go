package wallet

import (
	"time"

	"github.com/Lama189/soundwave-platform/billing-service/internal/domain"
	"github.com/google/uuid"
)

type GetWalletDTO struct {
	ArtistID       uuid.UUID
	TargetCurrency *domain.Currency
}

type WalletOutputDTO struct {
	ID                     uuid.UUID
	ArtistID               uuid.UUID
	Currency               domain.Currency
	Balance                domain.Money
	LockedBalance          domain.Money
	ConvertedBalance       *domain.Money
	ConvertedLockedBalance *domain.Money
	CreatedAt              time.Time
	UpdatedAt              time.Time
}

type CreditBalanceDTO struct {
	ArtistID      uuid.UUID
	Amount        domain.Money
	ReferenceType domain.ReferenceType
	ReferenceID   uuid.UUID
	Description   string
}

type CreditBalanceOutputDTO struct {
	WalletID      uuid.UUID
	NewBalance    domain.Money
	LedgerEntryID uuid.UUID
}

type GetStatementDTO struct {
	ArtistID uuid.UUID
	Limit    int
	Offset   int
}

type LedgerEntryOutputDTO struct {
	ID            uuid.UUID
	WalletID      uuid.UUID
	OperationType domain.LedgerOperationType
	Amount        domain.Money
	BalanceAfter  domain.Money
	ReferenceType domain.ReferenceType
	ReferenceID   uuid.UUID
	Description   string
	CreatedAt     time.Time
}

type StatementOutputDTO struct {
	WalletID uuid.UUID
	Entries  []*LedgerEntryOutputDTO
}

func toWalletOutputDTO(w *domain.Wallet, convertedBal, convertedLocked *domain.Money) *WalletOutputDTO {
	if w == nil {
		return nil
	}

	return &WalletOutputDTO{
		ID:                     w.ID,
		ArtistID:               w.ArtistID,
		Currency:               w.Currency,
		Balance:                w.Balance,
		LockedBalance:          w.LockedBalance,
		ConvertedBalance:       convertedBal,
		ConvertedLockedBalance: convertedLocked,
		CreatedAt:              w.CreatedAt,
		UpdatedAt:              w.UpdatedAt,
	}
}

func toLedgerEntryOutputDTO(e *domain.LedgerEntry) *LedgerEntryOutputDTO {
	if e == nil {
		return nil
	}

	return &LedgerEntryOutputDTO{
		ID:            e.ID,
		WalletID:      e.WalletID,
		OperationType: e.OperationType,
		Amount:        e.Amount,
		BalanceAfter:  e.BalanceAfter,
		ReferenceType: e.ReferenceType,
		ReferenceID:   e.ReferenceID,
		Description:   e.Description,
		CreatedAt:     e.CreatedAt,
	}
}

func toLedgerEntryOutputDTOList(entries []*domain.LedgerEntry) []*LedgerEntryOutputDTO {
	if entries == nil {
		return []*LedgerEntryOutputDTO{}
	}

	list := make([]*LedgerEntryOutputDTO, 0, len(entries))
	for _, e := range entries {
		list = append(list, toLedgerEntryOutputDTO(e))
	}
	return list
}
