package worker

import (
	"testing"

	"github.com/Lama189/soundwave-platform/order-service/internal/domain"
)

func TestSelectExchange(t *testing.T) {
	cases := []struct {
		eventType domain.EventType
		expected  string
	}{
		{domain.EventTypeCmdCatalogGrantEntitlements, OrderCommandsExchange},
		{domain.EventTypeCmdCatalogRevokeEntitlements, OrderCommandsExchange},
		{domain.EventTypeCmdBillingSettleOrder, OrderCommandsExchange},
		{domain.EventTypeCmdPaymentRefund, OrderCommandsExchange},
		{domain.EventTypeOrderFailed, OrderEventsExchange},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(string(tc.eventType), func(t *testing.T) {
			actual := selectExchange(tc.eventType)
			if actual != tc.expected {
				t.Fatalf("expected exchange %s for %s, got %s", tc.expected, tc.eventType, actual)
			}
		})
	}
}
