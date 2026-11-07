package domain

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

type ProcessedCommand struct {
	CommandID   uuid.UUID
	SagaID      uuid.UUID
	CommandName string
	CreatedAt   time.Time
}

func NewProcessedCommand(commandID, sagaID uuid.UUID, commandName string) (*ProcessedCommand, error) {
	cmd := &ProcessedCommand{
		CommandID:   commandID,
		SagaID:      sagaID,
		CommandName: commandName,
		CreatedAt:   time.Now().UTC(),
	}

	if err := cmd.Validate(); err != nil {
		return nil, fmt.Errorf("failed to create processed command: %w", err)
	}

	return cmd, nil
}

func RestoreProcessedCommand(commandID, sagaID uuid.UUID, commandName string, createdAt time.Time) *ProcessedCommand {
	return &ProcessedCommand{
		CommandID:   commandID,
		SagaID:      sagaID,
		CommandName: commandName,
		CreatedAt:   createdAt,
	}
}

func (c *ProcessedCommand) Validate() error {
	if c.CommandID == uuid.Nil {
		return fmt.Errorf("%w: command id is required", ErrInvalidInput)
	}
	if c.SagaID == uuid.Nil {
		return fmt.Errorf("%w: saga id is required", ErrInvalidInput)
	}
	if c.CommandName == "" {
		return fmt.Errorf("%w: command name is required", ErrInvalidInput)
	}

	return nil
}
