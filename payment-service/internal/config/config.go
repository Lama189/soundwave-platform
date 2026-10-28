package config

import "time"

type HTTPConfig struct {
	Port         string        `env:"HTTP_PORT" env-default:":8080"`
	ReadTimeout  time.Duration `env:"HTTP_READ_TIMEOUT" env-default:"5s"`
	WriteTimeout time.Duration `env:"HTTP_WRITE_TIMEOUT" env-default:"10s"`
	IdleTimeout  time.Duration `env:"HTTP_IDLE_TIMEOUT" env-default:"60s"`
}

type MockAdapterConfig struct {
	WebhookSecret string
}

type WebhookConfig struct {
	MaxWebhookBodySize int64 `env:"WEBHOOK_MAX_BODY_SIZE" env-default:"1048576"`
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
