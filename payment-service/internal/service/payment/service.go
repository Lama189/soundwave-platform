package payment

import (
	"context"
	"errors"
	"fmt"

	"github.com/Lama189/soundwave-platform/payment-service/internal/domain"
	"github.com/Lama189/soundwave-platform/payment-service/internal/grpcclient"
	"github.com/Lama189/soundwave-platform/payment-service/internal/uow"
	"github.com/google/uuid"
)

type Service struct {
	uow         uow.Manager
	orderClient OrderClient
	gateways    map[domain.PaymentProvider]PaymentGateway
}

func NewService(uow uow.Manager, orderClient OrderClient, gateways map[domain.PaymentProvider]PaymentGateway) *Service {
	return &Service{
		uow:         uow,
		orderClient: orderClient,
		gateways:    gateways,
	}
}

func (s *Service) CreatePaymentSession(ctx context.Context, input CreatePaymentSessionInput) (*PaymentSessionOutput, error) {
	if err := input.Validate(); err != nil {
		return nil, err
	}

	gateway, ok := s.gateways[input.Provider]
	if !ok {
		return nil, fmt.Errorf("%w: %s", domain.ErrUnsupportedProvider, input.Provider)
	}

	var existingPayment *domain.Payment
	err := s.uow.Do(ctx, func(work uow.UnitOfWork) error {
		p, err := work.Payments().GetByIdempotencyKey(ctx, input.IdempotencyKey)
		if err != nil {
			if errors.Is(err, domain.ErrNotFound) {
				return nil
			}
			return err
		}

		existingPayment = p
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("check idempotency key: %w", err)
	}

	if existingPayment != nil {
		if existingPayment.OrderID != input.OrderID || existingPayment.PayerUserID != input.PayerUserID {
			return nil, fmt.Errorf("%w: idempotency key %s already used for different order/user", domain.ErrConflict,
				input.IdempotencyKey)
		}
		return toPaymentSessionOutput(existingPayment, ""), nil
	}

	order, err := s.orderClient.GetOrder(ctx, grpcclient.GetOrderRequest{
		OrderID:     input.OrderID,
		BuyerUserID: &input.PayerUserID,
	})
	if err != nil {
		return nil, fmt.Errorf("fetch order %s: %w", input.OrderID, err)
	}

	if order.BuyerUserID != input.PayerUserID {
		return nil, fmt.Errorf("%w: user %s is not the buyer of order %s", domain.ErrForbidden, input.PayerUserID, input.OrderID)
	}

	if order.Status != "pending_payment" {
		if order.Status == "paid" {
			return nil, domain.ErrOrderAlreadyPaid
		}
		return nil, fmt.Errorf("%w: cannot pay order in status %q", domain.ErrConflict, order.Status)
	}

	payment, err := domain.NewPayment(
		order.ID,
		input.PayerUserID,
		order.TotalAmount,
		order.Currency,
		input.Provider,
		input.IdempotencyKey,
	)
	if err != nil {
		return nil, fmt.Errorf("create payment aggregate: %w", err)
	}

	session, err := gateway.CreateSession(ctx, domain.SessionParams{
		PaymentID:      payment.ID,
		OrderID:        payment.OrderID,
		Amount:         payment.Amount,
		Currency:       payment.Currency,
		CustomerEmail:  input.CustomerEmail,
		IdempotencyKey: input.IdempotencyKey,
	})
	if err != nil {
		return nil, fmt.Errorf("%w: %v", domain.ErrProviderCommunication, err)
	}

	if err := payment.MarkProcessing(session.ExternalID, session.ClientSecret); err != nil {
		return nil, fmt.Errorf("transition payment to processing: %w", err)
	}

	err = s.uow.Do(ctx, func(work uow.UnitOfWork) error {
		return work.Payments().Create(ctx, payment)
	})
	if err != nil {
		return nil, fmt.Errorf("save payment: %w", err)
	}

	output := toPaymentSessionOutput(payment, session.RedirectURL)
	output.ExpiresAt = session.ExpiresAt
	return output, nil
}

func (s *Service) HandleWebhook(ctx context.Context, input HandleWebhookInput) error {
	gateway, ok := s.gateways[input.Provider]
	if !ok {
		return fmt.Errorf("%w: %s", domain.ErrUnsupportedProvider, input.Provider)
	}

	normalizedEvent, err := gateway.ParseAndVerifyWebhook(ctx, input.Payload, input.Signature)
	if err != nil {
		return fmt.Errorf("verify and parse webhook: %w", err)
	}

	return s.uow.Do(ctx, func(work uow.UnitOfWork) error {
		existingEvent, err := work.WebhookEvents().GetByProviderAndExternalIDForUpdate(ctx, input.Provider, normalizedEvent.ExternalEventID)
		if err != nil && !errors.Is(err, domain.ErrNotFound) {
			return fmt.Errorf("check webhook event duplicate: %w", err)
		}

		if existingEvent != nil && existingEvent.IsTerminal() {
			return nil
		}

		var webhookEv *domain.WebhookEvent
		if existingEvent == nil {
			webhookEv, err = domain.NewWebhookEvent(input.Provider, normalizedEvent.ExternalEventID, string(normalizedEvent.Type), input.Payload)
			if err != nil {
				return fmt.Errorf("create webhook event entity: %w", err)
			}

			if err := work.WebhookEvents().Create(ctx, webhookEv); err != nil {
				return fmt.Errorf("persist webhook event: %w", err)
			}
		} else {
			webhookEv = existingEvent
		}

		if normalizedEvent.Type == domain.WebhookTypeIgnored {
			_ = webhookEv.MarkIgnored("event type not actionable")
			return work.WebhookEvents().Update(ctx, webhookEv)
		}

		payment, err := work.Payments().GetByExternalIDForUpdate(ctx, normalizedEvent.ExternalPaymentID)
		if err != nil {
			_ = webhookEv.MarkFailed(err)
			_ = work.WebhookEvents().Update(ctx, webhookEv)
			return fmt.Errorf("find payment by external_id %s: %w", normalizedEvent.ExternalPaymentID, err)
		}

		var outboxEvent *domain.Event

		switch normalizedEvent.Type {

		case domain.WebhookTypePaymentSucceeded:
			if err := payment.MarkSucceeded(normalizedEvent.ExternalPaymentID); err != nil {
				if errors.Is(err, domain.ErrPaymentAlreadyCompleted) {
					_ = webhookEv.MarkProcessed()
					return work.WebhookEvents().Update(ctx, webhookEv)
				}
				return fmt.Errorf("mark payment succeeded: %w", err)
			}

			outboxEvent, err = domain.NewEvent(
				domain.AggregateTypePayment,
				payment.ID,
				domain.EventTypePaymentSucceeded,
				domain.PaymentSucceededPayload{
					PaymentID:      payment.ID,
					OrderID:        payment.OrderID,
					PayerUserID:    payment.PayerUserID,
					Amount:         payment.Amount,
					Currency:       payment.Currency,
					Provider:       payment.Provider,
					ExternalID:     payment.ExternalID,
					IdempotencyKey: payment.IdempotencyKey,
				},
			)
			if err != nil {
				return fmt.Errorf("create payment.succeeded outbox event: %w", err)
			}

		case domain.WebhookTypePaymentFailed:
			if err := payment.MarkFailed(normalizedEvent.FailureReason, normalizedEvent.ExternalPaymentID); err != nil {
				if errors.Is(err, domain.ErrPaymentAlreadyCompleted) {
					_ = webhookEv.MarkProcessed()
					return work.WebhookEvents().Update(ctx, webhookEv)
				}
				return fmt.Errorf("mark payment failed: %w", err)
			}

			outboxEvent, err = domain.NewEvent(
				domain.AggregateTypePayment,
				payment.ID,
				domain.EventTypePaymentFailed,
				domain.PaymentFailedPayload{
					PaymentID:     payment.ID,
					OrderID:       payment.OrderID,
					FailureReason: normalizedEvent.FailureReason,
				},
			)
			if err != nil {
				return fmt.Errorf("create payment.failed outbox event: %w", err)
			}

		case domain.WebhookTypePaymentRefunded:
			if err := payment.MarkRefunded(); err != nil {
				if errors.Is(err, domain.ErrPaymentAlreadyCompleted) {
					_ = webhookEv.MarkProcessed()
					return work.WebhookEvents().Update(ctx, webhookEv)
				}
				return fmt.Errorf("mark payment refunded: %w", err)
			}

			outboxEvent, err = domain.NewEvent(
				domain.AggregateTypePayment,
				payment.ID,
				domain.EventTypePaymentRefunded,
				domain.PaymentRefundedPayload{
					PaymentID:  payment.ID,
					OrderID:    payment.OrderID,
					ExternalID: payment.ExternalID,
				},
			)
			if err != nil {
				return fmt.Errorf("create payment.refunded outbox event: %w", err)
			}
		}

		if err := work.Payments().Update(ctx, payment); err != nil {
			return fmt.Errorf("update payment status: %w", err)
		}

		if outboxEvent != nil {
			if err := work.Outbox().Record(ctx, outboxEvent); err != nil {
				return fmt.Errorf("record outbox event %s: %w", outboxEvent.EventType, err)
			}
		}

		_ = webhookEv.MarkProcessed()
		if err := work.WebhookEvents().Update(ctx, webhookEv); err != nil {
			return fmt.Errorf("update webhook event status: %w", err)
		}

		return nil
	})
}

func (s *Service) GetPaymentByID(ctx context.Context, paymentID uuid.UUID) (*PaymentOutput, error) {
	if paymentID == uuid.Nil {
		return nil, fmt.Errorf("%w: payment id is required", domain.ErrInvalidInput)
	}

	var payment *domain.Payment
	err := s.uow.Do(ctx, func(work uow.UnitOfWork) error {
		p, err := work.Payments().GetByID(ctx, paymentID)
		if err != nil {
			return err
		}

		payment = p
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("get payment by id %s: %w", paymentID, err)
	}

	return toPaymentOutput(payment), nil
}

func (s *Service) GetPaymentByOrderID(ctx context.Context, orderID uuid.UUID) (*PaymentOutput, error) {
	if orderID == uuid.Nil {
		return nil, fmt.Errorf("%w: order id is required", domain.ErrInvalidInput)
	}

	var payment *domain.Payment
	err := s.uow.Do(ctx, func(work uow.UnitOfWork) error {
		p, err := work.Payments().GetByOrderID(ctx, orderID)
		if err != nil {
			return err
		}
		payment = p
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("get payment by order id %s: %w", orderID, err)
	}

	return toPaymentOutput(payment), nil
}
