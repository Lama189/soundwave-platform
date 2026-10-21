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
	PollInterval time.Duration `env:"OUTBOX_RELAY_POLL_INTERVAL" env-default:"1s"`
	BatchSize    int           `env:"OUTBOX_RELAY_BATCH_SIZE" env-default:"50"`
}

func DefaultRelayConfig() RelayConfig {
	return RelayConfig{
		PollInterval: 1 * time.Second,
		BatchSize:    50,
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
