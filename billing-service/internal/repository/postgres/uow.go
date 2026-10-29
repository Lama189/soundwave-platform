package postgres

import (
	"context"
	"fmt"

	"github.com/Lama189/soundwave-platform/billing-service/internal/uow"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type pgxUnitOfWork struct {
	tx              pgx.Tx
	wallets         uow.WalletRepository
	ledger          uow.LedgerRepository
	payouts         uow.PayoutRepository
	outbox          uow.OutboxRepository
	processedEvents uow.ProcessedEventsRepository
}

func newPgxUnitOfWork(tx pgx.Tx) *pgxUnitOfWork {
	return &pgxUnitOfWork{tx: tx}
}

func (u *pgxUnitOfWork) Wallets() uow.WalletRepository {
	if u.wallets == nil {
		u.wallets = NewWalletRepository(u.tx)
	}
	return u.wallets
}

func (u *pgxUnitOfWork) Ledger() uow.LedgerRepository {
	if u.ledger == nil {
		u.ledger = NewLedgerRepository(u.tx)
	}
	return u.ledger
}

func (u *pgxUnitOfWork) Payouts() uow.PayoutRepository {
	if u.payouts == nil {
		u.payouts = NewPayoutRepository(u.tx)
	}
	return u.payouts
}

func (u *pgxUnitOfWork) Outbox() uow.OutboxRepository {
	if u.outbox == nil {
		u.outbox = NewOutboxRepository(u.tx)
	}
	return u.outbox
}

func (u *pgxUnitOfWork) ProcessedEvents() uow.ProcessedEventsRepository {
	if u.processedEvents == nil {
		u.processedEvents = NewProcessedEventsRepository(u.tx)
	}
	return u.processedEvents
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
