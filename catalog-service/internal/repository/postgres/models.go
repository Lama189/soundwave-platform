package postgres

import (
	"time"

	"github.com/Lama189/soundwave-platform/catalog-service/internal/domain"
	"github.com/google/uuid"
)

type artistModel struct {
	ID        uuid.UUID `db:"id"`
	UserID    uuid.UUID `db:"user_id"`
	Name      string    `db:"name"`
	Bio       string    `db:"bio"`
	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
}

func (m *artistModel) toDomain() *domain.Artist {
	return domain.RestoreArtist(
		m.ID,
		m.UserID,
		m.Name,
		m.Bio,
		m.CreatedAt,
		m.UpdatedAt,
	)
}

type albumModel struct {
	ID          uuid.UUID `db:"id"`
	ArtistID    uuid.UUID `db:"artist_id"`
	Title       string    `db:"title"`
	Description string    `db:"description"`
	CoverKey    string    `db:"cover_key"`
	CreatedAt   time.Time `db:"created_at"`
	UpdatedAt   time.Time `db:"updated_at"`
}

func (m *albumModel) toDomain() *domain.Album {
	return domain.RestoreAlbum(
		m.ID,
		m.ArtistID,
		m.Title,
		m.Description,
		m.CoverKey,
		m.CreatedAt,
		m.UpdatedAt,
	)
}

type trackModel struct {
	ID              uuid.UUID  `db:"id"`
	ArtistID        uuid.UUID  `db:"artist_id"`
	AlbumID         *uuid.UUID `db:"album_id"`
	Title           string     `db:"title"`
	DurationSeconds int        `db:"duration_seconds"`
	CoverKey        string     `db:"cover_key"`
	FileKey         string     `db:"file_key"`
	PreviewKey      string     `db:"preview_key"`
	RawKey          string     `db:"raw_key"`
	Status          string     `db:"status"`
	CreatedAt       time.Time  `db:"created_at"`
	UpdatedAt       time.Time  `db:"updated_at"`
}

func (m *trackModel) toDomain() *domain.Track {
	return domain.RestoreTrack(
		m.ID,
		m.ArtistID,
		m.AlbumID,
		m.Title,
		time.Duration(m.DurationSeconds)*time.Second,
		m.CoverKey,
		m.FileKey,
		m.PreviewKey,
		m.RawKey,
		domain.TrackStatus(m.Status),
		m.CreatedAt,
		m.UpdatedAt,
	)
}

type playlistModel struct {
	ID          uuid.UUID `db:"id"`
	UserID      uuid.UUID `db:"user_id"`
	Title       string    `db:"title"`
	Description string    `db:"description"`
	IsPrivate   bool      `db:"is_private"`
	CreatedAt   time.Time `db:"created_at"`
}

func (m *playlistModel) toDomain() *domain.Playlist {
	return domain.RestorePlaylist(
		m.ID,
		m.UserID,
		m.Title,
		m.Description,
		m.IsPrivate,
		m.CreatedAt,
	)
}

type playlistTrackModel struct {
	PlaylistID uuid.UUID `db:"playlist_id"`
	TrackID    uuid.UUID `db:"track_id"`
	Position   int       `db:"position"`
	AddedAt    time.Time `db:"added_at"`
}

func (m *playlistTrackModel) toDomain() *domain.PlaylistTrack {
	return domain.RestorePlaylistTrack(
		m.PlaylistID,
		m.TrackID,
		m.Position,
		m.AddedAt,
	)
}

type trackLikeModel struct {
	UserID    uuid.UUID `db:"user_id"`
	TrackID   uuid.UUID `db:"track_id"`
	CreatedAt time.Time `db:"created_at"`
}

func (m *trackLikeModel) toDomain() *domain.TrackLike {
	return domain.RestoreTrackLike(
		m.UserID,
		m.TrackID,
		m.CreatedAt,
	)
}

type listeningHistoryModel struct {
	ID         int64     `db:"id"`
	UserID     uuid.UUID `db:"user_id"`
	TrackID    uuid.UUID `db:"track_id"`
	ListenedAt time.Time `db:"listened_at"`
	Completed  bool      `db:"completed"`
}

func (m *listeningHistoryModel) toDomain() *domain.ListeningHistory {
	return domain.RestoreListeningHistory(
		m.ID,
		m.UserID,
		m.TrackID,
		m.ListenedAt,
		m.Completed,
	)
}

type EventModel struct {
	ID            uuid.UUID
	AggregateType string
	AggregateID   uuid.UUID
	EventType     string
	Payload       []byte
	CreatedAt     time.Time
	ProcessedAt   *time.Time
}

func (m *EventModel) toDomain() *domain.Event {
	return domain.RestoreEvent(
		m.ID,
		domain.AggregateType(m.AggregateType),
		m.AggregateID,
		domain.EventType(m.EventType),
		m.Payload,
		m.CreatedAt,
		m.ProcessedAt,
	)
}
