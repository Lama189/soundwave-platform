package postgres

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

type ProcessedCommandRepository struct {
	db DB
}

func NewProcessedCommandRepository(db DB) *ProcessedCommandRepository {
	return &ProcessedCommandRepository{db: db}
}

func (r *ProcessedCommandRepository) HasBeenProcessed(ctx context.Context, commandID uuid.UUID) (bool, error) {
	const query = `
		SELECT EXISTS (
			SELECT 1
			FROM processed_commands
			WHERE command_id = $1
		);
	`

	var exists bool
	if err := r.db.QueryRow(ctx, query, commandID).Scan(&exists); err != nil {
		return false, fmt.Errorf("check command processed: %w", mapError(err))
	}

	return exists, nil
}

func (r *ProcessedCommandRepository) MarkProcessed(ctx context.Context, commandID, sagaID uuid.UUID, commandName string) error {
	const query = `
		INSERT INTO processed_commands (command_id, saga_id, command_name, created_at)
		VALUES ($1, $2, $3, NOW())
		ON CONFLICT (command_id) DO NOTHING;
	`

	_, err := r.db.Exec(ctx, query, commandID, sagaID, commandName)
	if err != nil {
		return fmt.Errorf("mark command processed: %w", mapError(err))
	}

	return nil
}
