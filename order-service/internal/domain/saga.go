package domain

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

type SagaStatus string

const (
	SagaStatusIdle         SagaStatus = "IDLE"
	SagaStatusProcessing   SagaStatus = "PROCESSING"
	SagaStatusCompensating SagaStatus = "COMPENSATING"
	SagaStatusCompleted    SagaStatus = "COMPLETED"
	SagaStatusFailed       SagaStatus = "FAILED"
)

func (s SagaStatus) IsValid() bool {
	switch s {
	case SagaStatusIdle, SagaStatusProcessing, SagaStatusCompensating, SagaStatusCompleted, SagaStatusFailed:
		return true
	default:
		return false
	}
}

type SagaStep string

const (
	StepNone               SagaStep = "NONE"
	StepGrantEntitlements  SagaStep = "GRANT_ENTITLEMENTS"
	StepSettleBilling      SagaStep = "SETTLE_BILLING"
	StepRevokeEntitlements SagaStep = "REVOKE_ENTITLEMENTS"
	StepRefundPayment      SagaStep = "REFUND_PAYMENT"
	StepDone               SagaStep = "DONE"
	StepFailed             SagaStep = "FAILED"
)

func (s SagaStep) IsValid() bool {
	switch s {
	case StepNone, StepGrantEntitlements, StepSettleBilling, StepRevokeEntitlements, StepRefundPayment, StepDone, StepFailed:
		return true
	default:
		return false
	}
}

type SagaState struct {
	Status        SagaStatus
	Step          SagaStep
	StepRetries   int
	FailureReason *string
	StepTimeoutAt *time.Time
	UpdatedAt     time.Time
}

func InitialSagaState() SagaState {
	return SagaState{
		Status:      SagaStatusIdle,
		Step:        StepNone,
		StepRetries: 0,
		UpdatedAt:   time.Now().UTC(),
	}
}

func (o *Order) StartFulfillment(paymentID uuid.UUID, timeout time.Duration) error {
	if o == nil {
		return ErrNotFound
	}
	if paymentID == uuid.Nil {
		return fmt.Errorf("%w: payment id is required", ErrInvalidInput)
	}
	if timeout <= 0 {
		return fmt.Errorf("%w: timeout must be positive", ErrInvalidInput)
	}
	if o.Status != OrderStatusPendingPayment {
		return fmt.Errorf("%w: cannot start saga for order with status %s", ErrConflict, o.Status)
	}
	if o.Saga.Status != SagaStatusIdle {
		return fmt.Errorf("%w: saga is already started with status %s", ErrConflict, o.Saga.Status)
	}

	now := time.Now().UTC()
	timeoutAt := now.Add(timeout)

	o.PaymentID = &paymentID
	o.Status = OrderStatusProcessing
	o.Saga.Status = SagaStatusProcessing
	o.Saga.Step = StepGrantEntitlements
	o.Saga.StepRetries = 0
	o.Saga.StepTimeoutAt = &timeoutAt
	o.Saga.UpdatedAt = now
	o.UpdatedAt = now

	return nil
}

func (o *Order) OnEntitlementsGranted(timeout time.Duration) error {
	if o == nil {
		return ErrNotFound
	}
	if timeout <= 0 {
		return fmt.Errorf("%w: timeout must be positive", ErrInvalidInput)
	}
	if o.Saga.Status != SagaStatusProcessing || o.Saga.Step != StepGrantEntitlements {
		return fmt.Errorf("%w: cannot transition to settle billing from status %s and step %s",
			ErrConflict, o.Saga.Status, o.Saga.Step)
	}

	now := time.Now().UTC()
	timeoutAt := now.Add(timeout)

	o.Saga.Step = StepSettleBilling
	o.Saga.StepRetries = 0
	o.Saga.StepTimeoutAt = &timeoutAt
	o.Saga.UpdatedAt = now
	o.UpdatedAt = now

	return nil
}

func (o *Order) CompleteFulfillment() error {
	if o == nil {
		return ErrNotFound
	}
	if o.Saga.Status != SagaStatusProcessing || o.Saga.Step != StepSettleBilling {
		return fmt.Errorf("%w: cannot complete saga from status %s and step %s",
			ErrConflict, o.Saga.Status, o.Saga.Step)
	}

	now := time.Now().UTC()

	o.Status = OrderStatusPaid
	o.Saga.Status = SagaStatusCompleted
	o.Saga.Step = StepDone
	o.Saga.StepTimeoutAt = nil
	o.Saga.UpdatedAt = now
	o.UpdatedAt = now

	return nil
}

func (o *Order) OnEntitlementsGrantFailed(reason string, timeout time.Duration) error {
	if o == nil {
		return ErrNotFound
	}
	if timeout <= 0 {
		return fmt.Errorf("%w: timeout must be positive", ErrInvalidInput)
	}
	if o.Saga.Status != SagaStatusProcessing || o.Saga.Step != StepGrantEntitlements {
		return fmt.Errorf("%w: cannot fail grant entitlements from status %s and step %s",
			ErrConflict, o.Saga.Status, o.Saga.Step)
	}

	now := time.Now().UTC()
	timeoutAt := now.Add(timeout)

	o.Saga.Status = SagaStatusCompensating
	o.Saga.Step = StepRefundPayment
	o.Saga.StepRetries = 0
	o.Saga.StepRetries = 0
	if reason != "" {
		o.Saga.FailureReason = &reason
	}
	o.Saga.StepTimeoutAt = &timeoutAt
	o.Saga.UpdatedAt = now
	o.UpdatedAt = now

	return nil
}

func (o *Order) OnSettlementFailed(reason string, timeout time.Duration) error {
	if o == nil {
		return ErrNotFound
	}
	if timeout <= 0 {
		return fmt.Errorf("%w: timeout must be positive", ErrInvalidInput)
	}
	if o.Saga.Status != SagaStatusProcessing || o.Saga.Step !=
		StepSettleBilling {
		return fmt.Errorf("%w: cannot fail settlement from status %s and step %s",
			ErrConflict, o.Saga.Status, o.Saga.Step)
	}

	now := time.Now().UTC()
	timeoutAt := now.Add(timeout)

	o.Saga.Status = SagaStatusCompensating
	o.Saga.Step = StepRevokeEntitlements
	o.Saga.StepRetries = 0
	if reason != "" {
		o.Saga.FailureReason = &reason
	}
	o.Saga.StepTimeoutAt = &timeoutAt
	o.Saga.UpdatedAt = now
	o.UpdatedAt = now

	return nil
}

func (o *Order) OnEntitlementsRevoked(timeout time.Duration) error {
	if o == nil {
		return ErrNotFound
	}
	if timeout <= 0 {
		return fmt.Errorf("%w: timeout must be positive", ErrInvalidInput)
	}
	if o.Saga.Status != SagaStatusCompensating || o.Saga.Step !=
		StepRevokeEntitlements {
		return fmt.Errorf("%w: cannot transition to refund payment from status %s and step %s",
			ErrConflict, o.Saga.Status, o.Saga.Step)
	}

	now := time.Now().UTC()
	timeoutAt := now.Add(timeout)

	o.Saga.Step = StepRefundPayment
	o.Saga.StepRetries = 0
	o.Saga.StepTimeoutAt = &timeoutAt
	o.Saga.UpdatedAt = now
	o.UpdatedAt = now

	return nil
}

func (o *Order) CompleteCompensation() error {
	if o == nil {
		return ErrNotFound
	}
	if o.Saga.Status != SagaStatusCompensating || o.Saga.Step !=
		StepRefundPayment {
		return fmt.Errorf("%w: cannot complete compensation from status %s and step %s",
			ErrConflict, o.Saga.Status, o.Saga.Step)
	}

	now := time.Now().UTC()

	o.Status = OrderStatusCancelled
	o.Saga.Status = SagaStatusFailed
	o.Saga.Step = StepFailed
	o.Saga.StepTimeoutAt = nil
	o.Saga.UpdatedAt = now
	o.UpdatedAt = now

	return nil
}

func (o *Order) IsTimedOut(now time.Time) bool {
	if o == nil || o.Saga.StepTimeoutAt == nil {
		return false
	}
	if o.Saga.Status != SagaStatusProcessing && o.Saga.Status !=
		SagaStatusCompensating {
		return false
	}

	return now.After(*o.Saga.StepTimeoutAt)
}

func (o *Order) HandleTimeout(now time.Time, nextTimeout time.Duration, maxRetries int) (SagaStep, bool, error) {
	if !o.IsTimedOut(now) {
		return o.Saga.Step, false, fmt.Errorf("%w: order %s is not timed out", ErrConflict, o.ID)
	}

	if o.Saga.StepRetries < maxRetries {
		o.Saga.StepRetries++
		timeoutAt := now.Add(nextTimeout)
		o.Saga.StepTimeoutAt = &timeoutAt
		o.Saga.UpdatedAt = now
		o.UpdatedAt = now
		return o.Saga.Step, true, nil
	}

	o.Saga.StepRetries = 0

	switch o.Saga.Step {
	case StepGrantEntitlements:
		if err := o.OnEntitlementsGrantFailed("catalog step timeout after retries", nextTimeout); err != nil {
			return o.Saga.Step, false, err
		}
		return StepRefundPayment, false, nil

	case StepSettleBilling:
		if err := o.OnSettlementFailed("billing step timeout after retries", nextTimeout); err != nil {
			return o.Saga.Step, false, err
		}
		return StepRevokeEntitlements, false, nil

	case StepRevokeEntitlements:
		if err := o.OnEntitlementsRevoked(nextTimeout); err != nil {
			return o.Saga.Step, false, err
		}
		return StepRefundPayment, false, nil

	case StepRefundPayment:
		reason := "refund payment timeout after max retries, requires manual intervention"
		o.Saga.Status = SagaStatusFailed
		o.Saga.Step = StepFailed
		o.Status = OrderStatusCancelled
		o.Saga.FailureReason = &reason
		o.Saga.StepTimeoutAt = nil
		o.Saga.UpdatedAt = now
		o.UpdatedAt = now
		return StepFailed, false, nil

	default:
		return o.Saga.Step, false, fmt.Errorf("%w: unhandled timeout for saga step %s", ErrConflict, o.Saga.Step)
	}
}

func (o *Order) IsSagaActive() bool {
	if o == nil {
		return false
	}
	return o.Saga.Status == SagaStatusProcessing || o.Saga.Status == SagaStatusCompensating
}

func (o *Order) IsSagaTerminal() bool {
	if o == nil {
		return false
	}
	return o.Saga.Status == SagaStatusCompleted || o.Saga.Status == SagaStatusFailed
}
