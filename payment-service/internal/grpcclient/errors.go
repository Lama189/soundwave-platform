package grpcclient

import (
	"fmt"

	"github.com/Lama189/soundwave-platform/payment-service/internal/domain"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func mapGRPCError(err error) error {
	if err == nil {
		return nil
	}

	st, ok := status.FromError(err)
	if !ok {
		return fmt.Errorf("%w: %v", domain.ErrOrderCommunication, err)
	}

	switch st.Code() {
	case codes.NotFound:
		return fmt.Errorf("%w: %s", domain.ErrOrderNotFound, st.Message())

	case codes.AlreadyExists:
		return fmt.Errorf("%w: %s", domain.ErrOrderAlreadyPaid, st.Message())

	case codes.PermissionDenied, codes.Unauthenticated:
		return fmt.Errorf("%w: %s", domain.ErrForbidden, st.Message())

	case codes.InvalidArgument:
		return fmt.Errorf("%w: %s", domain.ErrInvalidInput, st.Message())

	case codes.DeadlineExceeded:
		return fmt.Errorf("%w: timeout: %s", domain.ErrOrderCommunication, st.Message())

	case codes.Canceled:
		return fmt.Errorf("%w: request canceled: %s", domain.ErrOrderCommunication, st.Message())

	case codes.Unavailable:
		return fmt.Errorf("%w: order service unavailable: %s", domain.ErrOrderCommunication, st.Message())

	default:
		return fmt.Errorf("%w: grpc code %s: %s", domain.ErrOrderCommunication, st.Code(), st.Message())
	}
}
