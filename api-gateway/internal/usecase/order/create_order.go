package order

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/Lama189/soundwave-platform/api-gateway/internal/grpcclient"
)

type CreateOrderUseCase struct {
	orderClient OrderClient
	logger      *slog.Logger
}

func NewCreateOrderUseCase(orderClient OrderClient, logger *slog.Logger) *CreateOrderUseCase {
	return &CreateOrderUseCase{
		orderClient: orderClient,
		logger:      logger,
	}
}

func (uc *CreateOrderUseCase) Execute(ctx context.Context, input CreateOrderInput) (*OrderOutput, error) {
	if err := input.Validate(); err != nil {
		return nil, err
	}

	items := make([]grpcclient.CreateOrderItemRequest, len(input.Items))
	for i, item := range input.Items {
		items[i] = grpcclient.CreateOrderItemRequest{
			ItemType: item.ItemType,
			ItemID:   item.ItemID,
		}
	}

	order, err := uc.orderClient.CreateOrder(ctx, grpcclient.CreateOrderRequest{
		BuyerUserID: input.BuyerUserID,
		Items:       items,
	})
	if err != nil {
		uc.logger.Warn(
			"failed to create order",
			slog.String("buyer_user_id", input.BuyerUserID.String()),
			slog.String("error", err.Error()),
		)
		return nil, fmt.Errorf("create order: %w", err)
	}

	return ToOrderOutput(order), nil
}
