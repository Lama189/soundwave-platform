package domain

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

type OrderStatus string

const (
	OrderStatusPendingPayment OrderStatus = "pending_payment"
	OrderStatusPaid           OrderStatus = "paid"
	OrderStatusFailed         OrderStatus = "failed"
	OrderStatusCancelled      OrderStatus = "cancelled"
	OrderStatusRefunded       OrderStatus = "refunded"
)

func (s OrderStatus) IsValid() bool {
	switch s {
	case OrderStatusCancelled, OrderStatusFailed, OrderStatusPaid, OrderStatusPendingPayment, OrderStatusRefunded:
		return true
	default:
		return false
	}
}

type Order struct {
	ID            uuid.UUID
	BuyerUserID   uuid.UUID
	Status        OrderStatus
	Items         []OrderItem
	TotalAmount   int64
	Currency      Currency
	PaymentID     *uuid.UUID
	FailureReason *string
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

func NewOrder(buyerUserID uuid.UUID, items []OrderItem) (*Order, error) {
	if len(items) == 0 {
		return nil, ErrEmptyOrderItems
	}

	orderCurrency := items[0].Currency
	var total int64
	for _, item := range items {
		if item.Currency != orderCurrency {
			return nil, fmt.Errorf("%w: items have different currencies (%s and %s)",
				ErrCurrencyMismatch, orderCurrency, item.Currency)
		}
		total += item.Price
	}

	now := time.Now().UTC()
	orderID := uuid.New()

	for i := range items {
		items[i].OrderID = orderID
	}

	o := &Order{
		ID:          orderID,
		BuyerUserID: buyerUserID,
		Status:      OrderStatusPendingPayment,
		Items:       items,
		TotalAmount: total,
		Currency:    orderCurrency,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := o.Validate(); err != nil {
		return nil, fmt.Errorf("failed to create order: %w", err)
	}

	return o, nil
}

func RestoreOrder(id, buyerUserID uuid.UUID, status OrderStatus, items []OrderItem, totalAmount int64, currency Currency, paymentID *uuid.UUID, failureReason *string, createdAt, updatedAt time.Time) *Order {
	return &Order{
		ID:            id,
		BuyerUserID:   buyerUserID,
		Status:        status,
		Items:         items,
		TotalAmount:   totalAmount,
		Currency:      currency,
		PaymentID:     paymentID,
		FailureReason: failureReason,
		CreatedAt:     createdAt,
		UpdatedAt:     updatedAt,
	}
}

func (o *Order) MarkPaid(paymentID uuid.UUID) error {
	if paymentID == uuid.Nil {
		return fmt.Errorf("%w: payment id is required", ErrInvalidInput)
	}

	if o.Status == OrderStatusPaid {
		return ErrOrderAlreadyPaid
	}
	if o.Status != OrderStatusPendingPayment {
		return fmt.Errorf("%w: cannot pay order with status %q", ErrOrderNotPending, o.Status)
	}

	o.Status = OrderStatusPaid
	o.PaymentID = &paymentID
	o.UpdatedAt = time.Now().UTC()

	return nil
}

func (o *Order) MarkFailed(reason string) error {
	if o.Status != OrderStatusPendingPayment {
		return fmt.Errorf("%w: cannot fail order with status %q", ErrOrderNotPending, o.Status)
	}

	o.Status = OrderStatusFailed
	if reason != "" {
		o.FailureReason = &reason
	}
	o.UpdatedAt = time.Now().UTC()

	return nil
}

func (o *Order) Cancel() error {
	if o.Status == OrderStatusPaid {
		return fmt.Errorf("%w: cannot cancel already paid order", ErrConflict)
	}
	if o.Status == OrderStatusCancelled {
		return nil
	}
	if o.Status != OrderStatusPendingPayment {
		return fmt.Errorf("%w: cannot cancel order with status %q", ErrConflict, o.Status)
	}

	o.Status = OrderStatusCancelled
	o.UpdatedAt = time.Now().UTC()

	return nil
}

func (o *Order) Validate() error {
	if o.ID == uuid.Nil {
		return fmt.Errorf("%w: order id is required", ErrInvalidInput)
	}
	if o.BuyerUserID == uuid.Nil {
		return fmt.Errorf("%w: buyer user id is required", ErrInvalidInput)
	}

	if !o.Status.IsValid() {
		return fmt.Errorf("%w: invalid order status: %s", ErrInvalidInput, o.Status)
	}

	if len(o.Items) == 0 {
		return ErrEmptyOrderItems
	}

	if o.TotalAmount <= 0 {
		return fmt.Errorf("%w: total amount must be positive", ErrInvalidInput)
	}

	if !o.Currency.IsValid() {
		return fmt.Errorf("%w: %q", ErrUnsupportedCurrency, o.Currency)
	}

	var calculatedTotal int64
	for _, item := range o.Items {
		if err := item.Validate(); err != nil {
			return fmt.Errorf("invalid order item: %w", err)
		}
		if item.Currency != o.Currency {
			return fmt.Errorf("%w: item currency %s != order currency %s",
				ErrCurrencyMismatch, item.Currency, o.Currency)
		}
		calculatedTotal += item.Price
	}

	if calculatedTotal != o.TotalAmount {
		return fmt.Errorf("%w: calculated items total %d != order total amount %d",
			ErrInvalidInput, calculatedTotal, o.TotalAmount)
	}

	return nil
}
