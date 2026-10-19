package postgres

import (
	"errors"
	"fmt"

	"github.com/Lama189/soundwave-platform/billing-service/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

const (
	errCodeUniqueViolation     = "23505"
	errCodeForeignKeyViolation = "23503"
	errCodeCheckViolation      = "23514"
)

func mapError(err error) error {
	if err == nil {
		return nil
	}

	if errors.Is(err, pgx.ErrNoRows) {
		return domain.ErrNotFound
	}

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case errCodeUniqueViolation:
			return fmt.Errorf("%w: %s (constraint: %s)", domain.ErrConflict, pgErr.Detail, pgErr.ConstraintName)
		case errCodeForeignKeyViolation:
			return fmt.Errorf("%w: %s (constraint: %s)", domain.ErrNotFound, pgErr.Detail, pgErr.ConstraintName)
		case errCodeCheckViolation:
			return fmt.Errorf("%w: %s (constraint: %s)", domain.ErrInvalidInput, pgErr.Detail, pgErr.ConstraintName)
		}
	}

	return err
}
