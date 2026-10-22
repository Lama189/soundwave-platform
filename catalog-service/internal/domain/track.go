package domain

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

type TrackStatus string

const (
	TrackStatusDraft      TrackStatus = "draft"
	TrackStatusProcessing TrackStatus = "processing"
	TrackStatusReady      TrackStatus = "ready"
	TrackStatusFailed     TrackStatus = "failed"
)

func (s TrackStatus) IsValid() bool {
	switch s {
	case TrackStatusDraft, TrackStatusProcessing, TrackStatusReady, TrackStatusFailed:
		return true
	default:
		return false
	}
}

var AllowedAudioExtensions = map[string]struct{}{
	".wav":  {},
	".mp3":  {},
	".flac": {},
	".ogg":  {},
	".aac":  {},
	".m4a":  {},
}

func IsSupportedAudioExtension(ext string) bool {
	_, ok := AllowedAudioExtensions[strings.ToLower(ext)]
	return ok
}

type Track struct {
	ID         uuid.UUID
	ArtistID   uuid.UUID
	AlbumID    *uuid.UUID
	Title      string
	Duration   time.Duration
	CoverKey   string
	FileKey    string
	PreviewKey string
	RawKey     string
	Price      int64
	Currency   Currency
	Status     TrackStatus
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

func NewTrack(artistID uuid.UUID, title string, price int64, currency Currency) (*Track, error) {
	if currency == "" {
		currency = CurrencyUSD
	}

	now := time.Now().UTC()
	t := &Track{
		ID:        uuid.New(),
		ArtistID:  artistID,
		Title:     strings.TrimSpace(title),
		Price:     price,
		Currency:  currency,
		Status:    TrackStatusDraft,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := t.Validate(); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrInvalidInput, err)
	}

	return t, nil
}

func RestoreTrack(
	id, artistID uuid.UUID,
	albumID *uuid.UUID,
	title string,
	duration time.Duration,
	coverKey, fileKey, previewKey, rawKey string,
	price int64,
	currency Currency,
	status TrackStatus,
	createdAt, updatedAt time.Time,
) *Track {
	return &Track{
		ID:         id,
		ArtistID:   artistID,
		AlbumID:    albumID,
		Title:      title,
		Duration:   duration,
		CoverKey:   coverKey,
		FileKey:    fileKey,
		PreviewKey: previewKey,
		RawKey:     rawKey,
		Price:      price,
		Currency:   currency,
		Status:     status,
		CreatedAt:  createdAt,
		UpdatedAt:  updatedAt,
	}
}

func (t *Track) SetRawKey(rawKey string) {
	t.RawKey = strings.TrimSpace(rawKey)
	t.UpdatedAt = time.Now().UTC()
}

func (t *Track) SetAlbum(albumID *uuid.UUID) {
	t.AlbumID = albumID
	t.UpdatedAt = time.Now().UTC()
}

func (t *Track) MarkAsReady(coverKey, fileKey, previewKey string, duration time.Duration) error {
	if t.Status != TrackStatusProcessing && t.Status != TrackStatusDraft {
		return fmt.Errorf("%w: cannot transition from %s to ready", ErrInvalidStateTransition, t.Status)
	}

	t.Status = TrackStatusReady
	t.FileKey = fileKey
	t.PreviewKey = previewKey
	t.Duration = duration

	if strings.TrimSpace(coverKey) != "" {
		t.CoverKey = strings.TrimSpace(coverKey)
	}

	t.UpdatedAt = time.Now().UTC()

	if err := t.Validate(); err != nil {
		return fmt.Errorf("%w: %w", ErrInvalidInput, err)
	}

	return nil
}

func (t *Track) MarkAsFailed() error {
	if t.Status == TrackStatusReady {
		return fmt.Errorf("%w: cannot fail track that is already ready", ErrInvalidStateTransition)
	}

	if t.Status == TrackStatusFailed {
		return nil
	}

	t.Status = TrackStatusFailed
	t.UpdatedAt = time.Now().UTC()

	return nil
}

func (t *Track) MarkAsProcessing() error {
	if t.Status != TrackStatusDraft {
		return fmt.Errorf("%w: cannot transition from %s to processing", ErrInvalidStateTransition, t.Status)
	}

	t.Status = TrackStatusProcessing
	t.UpdatedAt = time.Now().UTC()

	return nil
}

func (t *Track) SetPrice(price int64, currency Currency) error {
	t.Price = price
	t.Currency = currency
	t.UpdatedAt = time.Now().UTC()
	return t.Validate()
}

func (t *Track) Update(title string, albumID *uuid.UUID, coverKey string, price int64, currency Currency) error {
	t.Title = strings.TrimSpace(title)
	t.AlbumID = albumID
	if trimmedCoverKey := strings.TrimSpace(coverKey); trimmedCoverKey != "" {
		t.CoverKey = trimmedCoverKey
	}
	t.Price = price
	t.Currency = currency
	t.UpdatedAt = time.Now().UTC()

	return t.Validate()
}

func (t *Track) Validate() error {
	if t.ArtistID == uuid.Nil {
		return fmt.Errorf("artist ID cannot be empty")
	}

	trimmedTitle := strings.TrimSpace(t.Title)
	if trimmedTitle == "" {
		return fmt.Errorf("track title cannot be empty")
	}
	if len(trimmedTitle) > 255 {
		return fmt.Errorf("track title exceeds maximum length of 255 characters")
	}

	if t.Duration < 0 {
		return fmt.Errorf("track duration cannot be negative")
	}

	if t.Price < 0 {
		return fmt.Errorf("%w: track price cannot be negative, got %d", ErrNegativePrice, t.Price)
	}

	if !t.Currency.IsValid() {
		return fmt.Errorf("%w: %q", ErrUnsupportedCurrency, t.Currency)
	}

	if !t.Status.IsValid() {
		return fmt.Errorf("invalid track status: %s", t.Status)
	}

	if t.Status == TrackStatusReady {
		if t.FileKey == "" {
			return fmt.Errorf("ready track must have a file key")
		}
		if t.Duration == 0 {
			return fmt.Errorf("ready track must have duration greater than 0")
		}
	}

	if len(t.RawKey) > 512 {
		return fmt.Errorf("track raw key exceeds maximum length of 512 characters")
	}

	return nil
}
