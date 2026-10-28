package config

import "time"

type MockAdapterConfig struct {
	WebhookSecret string
}

type RelayConfig struct {
	PollInterval time.Duration `env:"OUTBOX_RELAY_POLL_INTERVAL" env-default:"1s"`
	BatchSize    int           `env:"OUTBOX_RELAY_BATCH_SIZE" env-default:"50"`
}

func DefaultRelayConfig() RelayConfig {
	return RelayConfig{
		PollInterval: 1 * time.Second,
		BatchSize:    50,
	}
}
