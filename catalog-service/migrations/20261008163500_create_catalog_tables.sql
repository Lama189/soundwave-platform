-- +goose Up
-- +goose StatementBegin
CREATE TYPE track_status AS ENUM ('draft', 'processing', 'ready', 'failed');

CREATE TABLE IF NOT EXISTS artists (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL UNIQUE,
    name VARCHAR(255) NOT NULL,
    bio TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_artists_user_id ON artists(user_id);

CREATE TABLE IF NOT EXISTS albums (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    artist_id UUID NOT NULL REFERENCES artists(id) ON DELETE CASCADE,
    title VARCHAR(255) NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    cover_key VARCHAR(512) NOT NULL DEFAULT '',
    price BIGINT NOT NULL DEFAULT 0,
    currency VARCHAR(3) NOT NULL DEFAULT 'USD',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_albums_price CHECK (price >= 0)
);

CREATE INDEX IF NOT EXISTS idx_albums_artist_id ON albums(artist_id);

CREATE TABLE IF NOT EXISTS tracks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    artist_id UUID NOT NULL REFERENCES artists(id) ON DELETE CASCADE,
    album_id UUID REFERENCES albums(id) ON DELETE SET NULL,
    title VARCHAR(255) NOT NULL,
    duration_seconds INT NOT NULL DEFAULT 0,
    cover_key VARCHAR(512) NOT NULL DEFAULT '',
    file_key VARCHAR(512) NOT NULL DEFAULT '',
    preview_key VARCHAR(512) NOT NULL DEFAULT '',
    raw_key VARCHAR(512) NOT NULL DEFAULT '',
    price BIGINT NOT NULL DEFAULT 0,
    currency VARCHAR(3) NOT NULL DEFAULT 'USD',
    status track_status NOT NULL DEFAULT 'draft',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_tracks_price CHECK (price >= 0)
);

CREATE INDEX IF NOT EXISTS idx_tracks_artist_id ON tracks(artist_id);
CREATE INDEX IF NOT EXISTS idx_tracks_album_id ON tracks(album_id);

CREATE TABLE IF NOT EXISTS playlists (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL,
    title VARCHAR(255) NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    is_private BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_playlists_user_id ON playlists(user_id);

CREATE TABLE IF NOT EXISTS playlist_tracks (
    playlist_id UUID NOT NULL REFERENCES playlists(id) ON DELETE CASCADE,
    track_id UUID NOT NULL REFERENCES tracks(id) ON DELETE CASCADE,
    position INT NOT NULL,
    added_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (playlist_id, track_id)
);

CREATE INDEX IF NOT EXISTS idx_playlist_tracks_order
    ON playlist_tracks(playlist_id, position ASC);

CREATE TABLE IF NOT EXISTS track_likes (
    user_id UUID NOT NULL,
    track_id UUID NOT NULL REFERENCES tracks(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, track_id)
);

CREATE INDEX IF NOT EXISTS idx_track_likes_user_id
    ON track_likes(user_id, created_at DESC);

CREATE TABLE IF NOT EXISTS listening_history (
    id BIGSERIAL PRIMARY KEY,
    user_id UUID NOT NULL,
    track_id UUID NOT NULL REFERENCES tracks(id) ON DELETE CASCADE,
    listened_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed BOOLEAN NOT NULL DEFAULT false
);

CREATE INDEX IF NOT EXISTS idx_listening_history_user_time
    ON listening_history(user_id, listened_at DESC);

CREATE TABLE IF NOT EXISTS outbox_events (
    id UUID PRIMARY KEY,
    aggregate_type VARCHAR(64) NOT NULL,
    aggregate_id UUID NOT NULL,
    event_type VARCHAR(64) NOT NULL,
    payload JSONB NOT NULL,
    headers JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    processed_at TIMESTAMPTZ,
    locked_until TIMESTAMPTZ,
    locked_by VARCHAR(64),
    retry_count INT NOT NULL DEFAULT 0,
    last_error TEXT
);

CREATE INDEX IF NOT EXISTS idx_outbox_unprocessed
    ON outbox_events (created_at ASC)
    INCLUDE (locked_until)
    WHERE processed_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_outbox_processed
    ON outbox_events (processed_at ASC)
    WHERE processed_at IS NOT NULL;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS outbox_events;
DROP TABLE IF EXISTS listening_history;
DROP TABLE IF EXISTS track_likes;
DROP TABLE IF EXISTS playlist_tracks;
DROP TABLE IF EXISTS playlists;
DROP TABLE IF EXISTS tracks;
DROP TABLE IF EXISTS albums;
DROP TABLE IF EXISTS artists;
DROP TYPE IF EXISTS track_status;
-- +goose StatementEnd