package postgres

import (
	"context"
	"fmt"

	"github.com/Lama189/soundwave-platform/payment-service/internal/uow"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type pgxUnitOfWork struct {
	tx            pgx.Tx
	payments      uow.PaymentRepository
	webhookEvents uow.WebhookEventRepository
	outbox        uow.OutboxRepository
}

func newPgxUnitOfWork(tx pgx.Tx) *pgxUnitOfWork {
	return &pgxUnitOfWork{tx: tx}
}

func (u *pgxUnitOfWork) Payments() uow.PaymentRepository {
	if u.payments == nil {
		u.payments = NewPaymentRepository(u.tx)
	}
	return u.payments
}

func (u *pgxUnitOfWork) WebhookEvents() uow.WebhookEventRepository {
	if u.webhookEvents == nil {
		u.webhookEvents = NewWebhookEventRepository(u.tx)
	}
	return u.webhookEvents
}

func (u *pgxUnitOfWork) Outbox() uow.OutboxRepository {
	if u.outbox == nil {
		u.outbox = NewOutboxRepository(u.tx)
	}
	return u.outbox
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
