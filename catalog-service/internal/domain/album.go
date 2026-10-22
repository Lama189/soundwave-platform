package domain

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

type Album struct {
	ID          uuid.UUID
	ArtistID    uuid.UUID
	Title       string
	Description string
	CoverKey    string
	Price       int64
	Currency    Currency
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func NewAlbum(artistID uuid.UUID, title, description, coverKey string, price int64, currency Currency) (*Album, error) {
	if currency == "" {
		currency = CurrencyUSD
	}

	now := time.Now().UTC()
	a := &Album{
		ArtistID:    artistID,
		Title:       strings.TrimSpace(title),
		Description: strings.TrimSpace(description),
		CoverKey:    strings.TrimSpace(coverKey),
		Price:       price,
		Currency:    currency,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := a.Validate(); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrInvalidInput, err)
	}

	return a, nil
}

func RestoreAlbum(
	id, artistID uuid.UUID,
	title, description, coverKey string,
	price int64,
	currency Currency,
	createdAt, updatedAt time.Time,
) *Album {
	return &Album{
		ID:          id,
		ArtistID:    artistID,
		Title:       title,
		Description: description,
		CoverKey:    coverKey,
		Price:       price,
		Currency:    currency,
		CreatedAt:   createdAt,
		UpdatedAt:   updatedAt,
	}
}

func (a *Album) Validate() error {
	if a.ArtistID == uuid.Nil {
		return fmt.Errorf("artist ID cannot be empty")
	}

	trimmedTitle := strings.TrimSpace(a.Title)
	if trimmedTitle == "" {
		return fmt.Errorf("album title cannot be empty")
	}
	if len(trimmedTitle) > 255 {
		return fmt.Errorf("album title exceeds maximum length of 255 characters")
	}

	if len(a.Description) > 2000 {
		return fmt.Errorf("album description exceeds maximum length of 2000 characters")
	}

	if len(a.CoverKey) > 512 {
		return fmt.Errorf("cover key exceeds maximum length of 512 characters")
	}

	if a.Price < 0 {
		return fmt.Errorf("%w: album price cannot be negative, got %d", ErrNegativePrice, a.Price)
	}

	if !a.Currency.IsValid() {
		return fmt.Errorf("%w: %q", ErrUnsupportedCurrency, a.Currency)
	}

	return nil
}

func (a *Album) SetPrice(price int64, currency Currency) error {
	a.Price = price
	a.Currency = currency
	a.UpdatedAt = time.Now().UTC()
	return a.Validate()
}

func (a *Album) Update(title, description, coverKey string, price int64, currency Currency) error {
	a.Title = strings.TrimSpace(title)
	a.Description = strings.TrimSpace(description)
	if trimmedCoverKey := strings.TrimSpace(coverKey); trimmedCoverKey != "" {
		a.CoverKey = trimmedCoverKey
	}
	a.Price = price
	a.Currency = currency
	a.UpdatedAt = time.Now().UTC()

	return a.Validate()
}
