package v1

import (
	"context"
	"strings"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	paymentpb "github.com/Lama189/soundwave-platform/gen/go/payment/v1"
	"github.com/Lama189/soundwave-platform/payment-service/internal/domain"
	"github.com/Lama189/soundwave-platform/payment-service/internal/service/payment"
)

type PaymentGRPCServer struct {
	paymentpb.UnimplementedPaymentServiceServer
	service PaymentService
}

func NewPaymentGRPCServer(service PaymentService) *PaymentGRPCServer {
	return &PaymentGRPCServer{
		service: service,
	}
}

func (s *PaymentGRPCServer) CreatePaymentSession(ctx context.Context, req *paymentpb.CreatePaymentSessionRequest) (*paymentpb.CreatePaymentSessionResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}

	if req.GetOrderId() == "" {
		return nil, status.Error(codes.InvalidArgument, "order id is required")
	}
	orderID, err := uuid.Parse(req.GetOrderId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid order id format")
	}

	if req.GetPayerUserId() == "" {
		return nil, status.Error(codes.InvalidArgument, "payer user id is required")
	}
	payerUserID, err := uuid.Parse(req.GetPayerUserId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid payer user id format")
	}

	provider := domain.PaymentProvider(req.GetProvider())
	if !provider.IsValid() {
		return nil, status.Errorf(codes.InvalidArgument, "invalid payment provider %q", req.GetProvider())
	}

	if strings.TrimSpace(req.GetIdempotencyKey()) == "" {
		return nil, status.Error(codes.InvalidArgument, "idempotency key is required")
	}

	session, err := s.service.CreatePaymentSession(ctx, payment.CreatePaymentSessionInput{
		OrderID:        orderID,
		PayerUserID:    payerUserID,
		Provider:       provider,
		CustomerEmail:  req.GetCustomerEmail(),
		IdempotencyKey: req.GetIdempotencyKey(),
	})
	if err != nil {
		return nil, mapDomainError(err)
	}

	return &paymentpb.CreatePaymentSessionResponse{
		Session: toProtoPaymentSession(session),
	}, nil
}

func (s *PaymentGRPCServer) GetPayment(ctx context.Context, req *paymentpb.GetPaymentRequest) (*paymentpb.GetPaymentResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}

	if req.GetPaymentId() == "" {
		return nil, status.Error(codes.InvalidArgument, "payment id is required")
	}
	paymentID, err := uuid.Parse(req.GetPaymentId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid payment id format")
	}

	p, err := s.service.GetPaymentByID(ctx, paymentID)
	if err != nil {
		return nil, mapDomainError(err)
	}

	return &paymentpb.GetPaymentResponse{
		Payment: toProtoPayment(p),
	}, nil
}

func (s *PaymentGRPCServer) GetPaymentByOrder(ctx context.Context, req *paymentpb.GetPaymentByOrderRequest) (*paymentpb.GetPaymentByOrderResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}

	if req.GetOrderId() == "" {
		return nil, status.Error(codes.InvalidArgument, "order id is required")
	}
	orderID, err := uuid.Parse(req.GetOrderId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid order id format")
	}

	p, err := s.service.GetPaymentByOrderID(ctx, orderID)
	if err != nil {
		return nil, mapDomainError(err)
	}

	return &paymentpb.GetPaymentByOrderResponse{
		Payment: toProtoPayment(p),
	}, nil
}

func toProtoPaymentSession(s *payment.PaymentSessionOutput) *paymentpb.PaymentSession {
	if s == nil {
		return nil
	}

	var expiresAt *timestamppb.Timestamp
	if s.ExpiresAt != nil {
		expiresAt = timestamppb.New(*s.ExpiresAt)
	}

	return &paymentpb.PaymentSession{
		PaymentId:    s.PaymentID.String(),
		OrderId:      s.OrderID.String(),
		Status:       string(s.Status),
		Provider:     string(s.Provider),
		ExternalId:   s.ExternalID,
		ClientSecret: s.ClientSecret,
		RedirectUrl:  s.RedirectURL,
		Amount:       s.Amount,
		Currency:     string(s.Currency),
		ExpiresAt:    expiresAt,
	}
}

func toProtoPayment(p *payment.PaymentOutput) *paymentpb.Payment {
	if p == nil {
		return nil
	}

	return &paymentpb.Payment{
		Id:             p.ID.String(),
		OrderId:        p.OrderID.String(),
		PayerUserId:    p.PayerUserID.String(),
		Amount:         p.Amount,
		Currency:       string(p.Currency),
		Status:         string(p.Status),
		Provider:       string(p.Provider),
		ExternalId:     p.ExternalID,
		ClientSecret:   p.ClientSecret,
		FailureReason:  p.FailureReason,
		IdempotencyKey: p.IdempotencyKey,
		CreatedAt:      timestamppb.New(p.CreatedAt),
		UpdatedAt:      timestamppb.New(p.UpdatedAt),
	}
}
