package postgres

import (
	"context"
	"fmt"

	"github.com/Lama189/soundwave-platform/catalog-service/internal/uow"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type pgxUnitOfWork struct {
	tx      pgx.Tx
	tracks  uow.TrackRepository
	outbox  uow.OutboxRepository
	artists uow.ArtistRepository
	albums  uow.AlbumRepository
}

func newPgxUnitOfWork(tx pgx.Tx) *pgxUnitOfWork {
	return &pgxUnitOfWork{tx: tx}
}

func (u *pgxUnitOfWork) Tracks() uow.TrackRepository {
	if u.tracks == nil {
		u.tracks = NewTrackRepository(u.tx)
	}
	return u.tracks
}

func (u *pgxUnitOfWork) Outbox() uow.OutboxRepository {
	if u.outbox == nil {
		u.outbox = NewOutboxRepository(u.tx)
	}
	return u.outbox
}

func (u *pgxUnitOfWork) Artists() uow.ArtistRepository {
	if u.artists == nil {
		u.artists = NewArtistRepository(u.tx)
	}
	return u.artists
}

func (u *pgxUnitOfWork) Albums() uow.AlbumRepository {
	if u.albums == nil {
		u.albums = NewAlbumRepository(u.tx)
	}
	return u.albums
}

type UnitOfWorkManager struct {
	pool *pgxpool.Pool
}

func NewUnitOfWorkManager(pool *pgxpool.Pool) *UnitOfWorkManager {
	return &UnitOfWorkManager{pool: pool}
}

func (m *UnitOfWorkManager) Do(ctx context.Context, fn func(uow uow.UnitOfWork) error) (err error) {
	tx, err := m.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}

	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback(ctx)
			panic(p)
		} else if err != nil {
			_ = tx.Rollback(ctx)
		}
	}()

	work := newPgxUnitOfWork(tx)
	if err = fn(work); err != nil {
		return err
	}

	if err = tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit tx: %w", err)
	}

	return nil
}
