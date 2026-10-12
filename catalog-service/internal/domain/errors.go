package domain

import (
	"errors"
	"fmt"
)

// Base errors
var (
	ErrNotFound      = errors.New("not found")
	ErrAlreadyExists = errors.New("already exists")
	ErrConflict      = errors.New("conflict")
	ErrInvalidInput  = errors.New("invalid input")
	ErrForbidden     = errors.New("permission denied")
)

// Artist errors
var (
	ErrArtistNotFound      = fmt.Errorf("%w: artist not found", ErrNotFound)
	ErrArtistAlreadyExists = fmt.Errorf("%w: user already has an artist profile", ErrAlreadyExists)
	ErrNotArtistOwner      = fmt.Errorf("%w: you are not the owner of this artist profile", ErrForbidden)
)

// Album errors
var (
	ErrAlbumNotFound = fmt.Errorf("%w: album not found", ErrNotFound)
	ErrNotAlbumOwner = fmt.Errorf("%w: you are not the owner of this album", ErrForbidden)
)

// Track errors
var (
	ErrTrackNotFound          = fmt.Errorf("%w: track not found", ErrNotFound)
	ErrTrackNotReady          = fmt.Errorf("%w: track is not ready yet", ErrInvalidInput)
	ErrInvalidAudioDuration   = fmt.Errorf("%w: duration must be positive", ErrInvalidInput)
	ErrInvalidStateTransition = fmt.Errorf("%w: invalid state transition", ErrConflict)
	ErrUnsupportedAudioFormat = fmt.Errorf("%w: unsupported audio format", ErrInvalidInput)
	ErrRawFileNotFound        = fmt.Errorf("%w: raw audio file not found in storage", ErrNotFound)
)

// Playlist errors
var (
	ErrPlaylistNotFound       = fmt.Errorf("%w: playlist not found", ErrNotFound)
	ErrNotPlaylistOwner       = fmt.Errorf("%w: you are not the owner of this playlist", ErrForbidden)
	ErrTrackAlreadyInPlaylist = fmt.Errorf("%w: track already in playlist", ErrAlreadyExists)
	ErrTrackNotInPlaylist     = fmt.Errorf("%w: track not found in playlist", ErrNotFound)
	ErrPrivatePlaylistAccess  = fmt.Errorf("%w: playlist is private", ErrForbidden)
)

// Like errors
var (
	ErrTrackAlreadyLiked = fmt.Errorf("%w: track is already liked", ErrAlreadyExists)
	ErrTrackNotLiked     = fmt.Errorf("%w: track is not liked", ErrNotFound)
)
