package config

import (
	"fmt"
	"os"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
)

type Config struct {
	Env      string `env:"ENV" env-default:"local"`
	GRPC     GRPCConfig
	Postgres PostgresConfig
	RabbitMQ RabbitMQConfig
	Relay    RelayConfig
	Catalog  CatalogConfig
}

type CatalogConfig struct {
	URL     string        `env:"CATALOG_SERVICE_URL" env-default:"localhost:50052"`
	Timeout time.Duration `env:"CATALOG_SERVICE_TIMEOUT" env-default:"5s"`
}

type GRPCConfig struct {
	Port string `env:"GRPC_PORT" env-default:":50054"`
}

type PostgresConfig struct {
	URL string `env:"DATABASE_URL" env-required:"true"`
}

type RabbitMQConfig struct {
	URL string `env:"RABBITMQ_URL" env-default:"amqp://guest:guest@localhost:5672"`
}

type RelayConfig struct {
	PollInterval  time.Duration `env:"OUTBOX_RELAY_POLL_INTERVAL" env-default:"1s"`
	BatchSize     int           `env:"OUTBOX_RELAY_BATCH_SIZE" env-default:"50"`
	LockDuration  time.Duration `env:"OUTBOX_RELAY_LOCK_DURATION" env-default:"30s"`
	MaxRetries    int           `env:"OUTBOX_RELAY_MAX_RETRIES" env-default:"5"`
	RetentionDays int           `env:"OUTBOX_RELAY_RETENTION_DAYS" env-default:"7"`
}

func DefaultRelayConfig() RelayConfig {
	return RelayConfig{
		PollInterval:  1 * time.Second,
		BatchSize:     50,
		LockDuration:  30 * time.Second,
		MaxRetries:    5,
		RetentionDays: 7,
	}
}

func Load() (*Config, error) {
	var cfg Config

	if _, err := os.Stat(".env"); err == nil {
		if err := cleanenv.ReadConfig(".env", &cfg); err != nil {
			return nil, fmt.Errorf("failed to read .env file: %w", err)
		}
		return &cfg, nil
	}

	if err := cleanenv.ReadEnv(&cfg); err != nil {
		return nil, fmt.Errorf("failed to read env variables: %w", err)
	}

	return &cfg, nil
}

func MustLoad() *Config {
	cfg, err := Load()
	if err != nil {
		panic(fmt.Sprintf("config error: %s", err))
	}
	return cfg
}
