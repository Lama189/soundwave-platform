package domain

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

type Playlist struct {
	ID          uuid.UUID
	UserID      uuid.UUID
	Title       string
	Description string
	IsPrivate   bool
	CreatedAt   time.Time
}

func NewPlaylist(userID uuid.UUID, title, description string, isPrivate bool) (*Playlist, error) {
	now := time.Now().UTC()
	p := &Playlist{
		UserID:      userID,
		Title:       strings.TrimSpace(title),
		Description: strings.TrimSpace(description),
		IsPrivate:   isPrivate,
		CreatedAt:   now,
	}

	if err := p.Validate(); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrInvalidInput, err)
	}

	return p, nil
}

func RestorePlaylist(id, userID uuid.UUID, title, description string, isPrivate bool, createdAt time.Time) *Playlist {
	return &Playlist{
		ID:          id,
		UserID:      userID,
		Title:       title,
		Description: description,
		IsPrivate:   isPrivate,
		CreatedAt:   createdAt,
	}
}

func (p *Playlist) Validate() error {
	if p.UserID == uuid.Nil {
		return fmt.Errorf("user ID cannot be empty")
	}

	trimmedTitle := strings.TrimSpace(p.Title)
	if trimmedTitle == "" {
		return fmt.Errorf("playlist title cannot be empty")
	}
	if len(trimmedTitle) > 255 {
		return fmt.Errorf("playlist title exceeds maximum length of 255 characters")
	}

	if len(p.Description) > 2000 {
		return fmt.Errorf("playlist description exceeds maximum length of 2000 characters")
	}

	return nil
}

func (p *Playlist) Update(title, description string, isPrivate bool) error {
	p.Title = strings.TrimSpace(title)
	p.Description = strings.TrimSpace(description)
	p.IsPrivate = isPrivate

	return p.Validate()
}

type PlaylistTrack struct {
	PlaylistID uuid.UUID
	TrackID    uuid.UUID
	Position   int
	AddedAt    time.Time
}

func NewPlaylistTrack(playlistID, trackID uuid.UUID, position int) (*PlaylistTrack, error) {
	now := time.Now().UTC()
	pt := &PlaylistTrack{
		PlaylistID: playlistID,
		TrackID:    trackID,
		Position:   position,
		AddedAt:    now,
	}

	if err := pt.Validate(); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrInvalidInput, err)
	}

	return pt, nil
}

func RestorePlaylistTrack(playlistID, trackID uuid.UUID, position int, addedAt time.Time) *PlaylistTrack {
	return &PlaylistTrack{
		PlaylistID: playlistID,
		TrackID:    trackID,
		Position:   position,
		AddedAt:    addedAt,
	}
}

func (pt *PlaylistTrack) Validate() error {
	if pt.PlaylistID == uuid.Nil {
		return fmt.Errorf("playlist ID cannot be empty")
	}
	if pt.TrackID == uuid.Nil {
		return fmt.Errorf("track ID cannot be empty")
	}
	if pt.Position < 0 {
		return fmt.Errorf("playlist track position cannot be negative")
	}

	return nil
}
