package payout

import (
	"time"

	"github.com/google/uuid"

	"github.com/Lama189/soundwave-platform/billing-service/internal/domain"
)

type RequestPayoutDTO struct {
	ArtistID        uuid.UUID
	Amount          domain.Money
	DestinationCard string
}

type CompletePayoutDTO struct {
	PayoutID uuid.UUID
}

type FailPayoutDTO struct {
	PayoutID uuid.UUID
	Reason   string
}

type ListPayoutsDTO struct {
	ArtistID uuid.UUID
	Status   *domain.PayoutStatus
	Limit    int
	Offset   int
}

type PayoutOutputDTO struct {
	ID              uuid.UUID
	ArtistID        uuid.UUID
	WalletID        uuid.UUID
	Amount          domain.Money
	DestinationCard string
	Status          string
	FailureReason   *string
	RequestedAt     time.Time
	ProcessedAt     *time.Time
}

type PayoutListOutputDTO struct {
	Payouts []*PayoutOutputDTO
	Total   int
}

func toPayoutOutputDTO(p *domain.Payout) *PayoutOutputDTO {
	if p == nil {
		return nil
	}
	return &PayoutOutputDTO{
		ID:              p.ID,
		ArtistID:        p.ArtistID,
		WalletID:        p.WalletID,
		Amount:          p.Amount,
		DestinationCard: p.DestinationCard,
		Status:          string(p.Status),
		FailureReason:   p.FailureReason,
		RequestedAt:     p.RequestedAt,
		ProcessedAt:     p.ProcessedAt,
	}
}

func toPayoutOutputDTOList(payouts []*domain.Payout) []*PayoutOutputDTO {
	if payouts == nil {
		return []*PayoutOutputDTO{}
	}
	list := make([]*PayoutOutputDTO, 0, len(payouts))
	for _, p := range payouts {
		list = append(list, toPayoutOutputDTO(p))
	}
	return list
}
