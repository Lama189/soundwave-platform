package v1

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	paymentpb "github.com/Lama189/soundwave-platform/gen/go/payment/v1"
	"github.com/Lama189/soundwave-platform/payment-service/internal/domain"
	"github.com/Lama189/soundwave-platform/payment-service/internal/service/payment"
)

type mockPaymentService struct {
	createSessionFn  func(ctx context.Context, input payment.CreatePaymentSessionInput) (*payment.PaymentSessionOutput, error)
	getByIDFn        func(ctx context.Context, paymentID uuid.UUID) (*payment.PaymentOutput, error)
	getByOrderIDFn    func(ctx context.Context, orderID uuid.UUID) (*payment.PaymentOutput, error)
}

func (m *mockPaymentService) CreatePaymentSession(ctx context.Context, input payment.CreatePaymentSessionInput) (*payment.PaymentSessionOutput, error) {
	if m.createSessionFn != nil {
		return m.createSessionFn(ctx, input)
	}
	return nil, errors.New("not implemented")
}

func (m *mockPaymentService) GetPaymentByID(ctx context.Context, paymentID uuid.UUID) (*payment.PaymentOutput, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, paymentID)
	}
	return nil, errors.New("not implemented")
}

func (m *mockPaymentService) GetPaymentByOrderID(ctx context.Context, orderID uuid.UUID) (*payment.PaymentOutput, error) {
	if m.getByOrderIDFn != nil {
		return m.getByOrderIDFn(ctx, orderID)
	}
	return nil, errors.New("not implemented")
}

func TestPaymentGRPCServer_CreatePaymentSession(t *testing.T) {
	orderID := uuid.New()
	payerUserID := uuid.New()
	paymentID := uuid.New()
	now := time.Now()

	tests := []struct {
		name          string
		req           *paymentpb.CreatePaymentSessionRequest
		mockFn        func(ctx context.Context, input payment.CreatePaymentSessionInput) (*payment.PaymentSessionOutput, error)
		expectedCode  codes.Code
		expectSuccess bool
	}{
		{
			name:         "nil request",
			req:          nil,
			expectedCode: codes.InvalidArgument,
		},
		{
			name: "missing order id",
			req: &paymentpb.CreatePaymentSessionRequest{
				OrderId:        "",
				PayerUserId:    payerUserID.String(),
				Provider:       "mock",
				IdempotencyKey: "key-1",
			},
			expectedCode: codes.InvalidArgument,
		},
		{
			name: "invalid order id format",
			req: &paymentpb.CreatePaymentSessionRequest{
				OrderId:        "invalid-uuid",
				PayerUserId:    payerUserID.String(),
				Provider:       "mock",
				IdempotencyKey: "key-1",
			},
			expectedCode: codes.InvalidArgument,
		},
		{
			name: "missing payer user id",
			req: &paymentpb.CreatePaymentSessionRequest{
				OrderId:        orderID.String(),
				PayerUserId:    "",
				Provider:       "mock",
				IdempotencyKey: "key-1",
			},
			expectedCode: codes.InvalidArgument,
		},
		{
			name: "invalid provider",
			req: &paymentpb.CreatePaymentSessionRequest{
				OrderId:        orderID.String(),
				PayerUserId:    payerUserID.String(),
				Provider:       "unknown-provider",
				IdempotencyKey: "key-1",
			},
			expectedCode: codes.InvalidArgument,
		},
		{
			name: "missing idempotency key",
			req: &paymentpb.CreatePaymentSessionRequest{
				OrderId:        orderID.String(),
				PayerUserId:    payerUserID.String(),
				Provider:       "mock",
				IdempotencyKey: "   ",
			},
			expectedCode: codes.InvalidArgument,
		},
		{
			name: "service domain error mapped correctly",
			req: &paymentpb.CreatePaymentSessionRequest{
				OrderId:        orderID.String(),
				PayerUserId:    payerUserID.String(),
				Provider:       "mock",
				IdempotencyKey: "key-1",
			},
			mockFn: func(ctx context.Context, input payment.CreatePaymentSessionInput) (*payment.PaymentSessionOutput, error) {
				return nil, domain.ErrConflict
			},
			expectedCode: codes.AlreadyExists,
		},
		{
			name: "success",
			req: &paymentpb.CreatePaymentSessionRequest{
				OrderId:        orderID.String(),
				PayerUserId:    payerUserID.String(),
				Provider:       "mock",
				CustomerEmail:  "test@example.com",
				IdempotencyKey: "key-1",
			},
			mockFn: func(ctx context.Context, input payment.CreatePaymentSessionInput) (*payment.PaymentSessionOutput, error) {
				return &payment.PaymentSessionOutput{
					PaymentID:    paymentID,
					OrderID:      input.OrderID,
					Status:       domain.PaymentStatusPending,
					Provider:     input.Provider,
					ExternalID:   "ext-123",
					ClientSecret: "secret-123",
					RedirectURL:  "https://example.com/pay",
					Amount:       1500,
					Currency:     "RUB",
					ExpiresAt:    &now,
				}, nil
			},
			expectSuccess: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSvc := &mockPaymentService{createSessionFn: tt.mockFn}
			server := NewPaymentGRPCServer(mockSvc)

			resp, err := server.CreatePaymentSession(context.Background(), tt.req)
			if tt.expectSuccess {
				if err != nil {
					t.Fatalf("expected success, got error: %v", err)
				}
				if resp == nil || resp.Session == nil {
					t.Fatal("expected non-nil response session")
				}
				if resp.Session.PaymentId != paymentID.String() {
					t.Errorf("expected payment id %s, got %s", paymentID, resp.Session.PaymentId)
				}
				if resp.Session.RedirectUrl != "https://example.com/pay" {
					t.Errorf("expected redirect url, got %s", resp.Session.RedirectUrl)
				}
			} else {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				st, ok := status.FromError(err)
				if !ok {
					t.Fatalf("expected grpc status error, got: %v", err)
				}
				if st.Code() != tt.expectedCode {
					t.Errorf("expected status code %v, got %v", tt.expectedCode, st.Code())
				}
			}
		})
	}
}

func TestPaymentGRPCServer_GetPayment(t *testing.T) {
	paymentID := uuid.New()
	orderID := uuid.New()
	payerUserID := uuid.New()
	now := time.Now()

	server := NewPaymentGRPCServer(&mockPaymentService{
		getByIDFn: func(ctx context.Context, id uuid.UUID) (*payment.PaymentOutput, error) {
			if id == paymentID {
				return &payment.PaymentOutput{
					ID:             id,
					OrderID:        orderID,
					PayerUserID:    payerUserID,
					Amount:         500,
					Currency:       "RUB",
					Status:         domain.PaymentStatusSucceeded,
					Provider:       domain.PaymentProviderMock,
					ExternalID:     "ext-1",
					ClientSecret:   "sec-1",
					IdempotencyKey: "key-1",
					CreatedAt:      now,
					UpdatedAt:      now,
				}, nil
			}
			return nil, domain.ErrNotFound
		},
	})

	t.Run("invalid uuid", func(t *testing.T) {
		_, err := server.GetPayment(context.Background(), &paymentpb.GetPaymentRequest{PaymentId: "bad-id"})
		if err == nil || status.Code(err) != codes.InvalidArgument {
			t.Errorf("expected InvalidArgument, got %v", err)
		}
	})

	t.Run("not found", func(t *testing.T) {
		_, err := server.GetPayment(context.Background(), &paymentpb.GetPaymentRequest{PaymentId: uuid.New().String()})
		if err == nil || status.Code(err) != codes.NotFound {
			t.Errorf("expected NotFound, got %v", err)
		}
	})

	t.Run("success", func(t *testing.T) {
		resp, err := server.GetPayment(context.Background(), &paymentpb.GetPaymentRequest{PaymentId: paymentID.String()})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.Payment.Id != paymentID.String() {
			t.Errorf("expected payment id %s, got %s", paymentID, resp.Payment.Id)
		}
		if resp.Payment.Status != string(domain.PaymentStatusSucceeded) {
			t.Errorf("expected status %s, got %s", domain.PaymentStatusSucceeded, resp.Payment.Status)
		}
	})
}

func TestPaymentGRPCServer_GetPaymentByOrder(t *testing.T) {
	orderID := uuid.New()
	paymentID := uuid.New()
	payerUserID := uuid.New()
	now := time.Now()

	server := NewPaymentGRPCServer(&mockPaymentService{
		getByOrderIDFn: func(ctx context.Context, id uuid.UUID) (*payment.PaymentOutput, error) {
			if id == orderID {
				return &payment.PaymentOutput{
					ID:             paymentID,
					OrderID:        orderID,
					PayerUserID:    payerUserID,
					Amount:         1000,
					Currency:       "RUB",
					Status:         domain.PaymentStatusPending,
					Provider:       domain.PaymentProviderMock,
					ExternalID:     "ext-2",
					ClientSecret:   "sec-2",
					IdempotencyKey: "key-2",
					CreatedAt:      now,
					UpdatedAt:      now,
				}, nil
			}
			return nil, domain.ErrNotFound
		},
	})

	t.Run("invalid uuid", func(t *testing.T) {
		_, err := server.GetPaymentByOrder(context.Background(), &paymentpb.GetPaymentByOrderRequest{OrderId: "bad-id"})
		if err == nil || status.Code(err) != codes.InvalidArgument {
			t.Errorf("expected InvalidArgument, got %v", err)
		}
	})

	t.Run("success", func(t *testing.T) {
		resp, err := server.GetPaymentByOrder(context.Background(), &paymentpb.GetPaymentByOrderRequest{OrderId: orderID.String()})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.Payment.OrderId != orderID.String() {
			t.Errorf("expected order id %s, got %s", orderID, resp.Payment.OrderId)
		}
	})
}
