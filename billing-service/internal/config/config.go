package config

import (
	"fmt"
	"os"
	"time"

	"github.com/Lama189/soundwave-platform/billing-service/internal/domain"
	"github.com/ilyakaznacheev/cleanenv"
)

type Config struct {
	Env      string `env:"ENV" env-default:"local"`
	GRPC     GRPCConfig
	Postgres PostgresConfig
	Redis    RedisConfig
	RabbitMQ RabbitMQConfig
	FX       FXConfig
	Billing  BillingConfig
	Relay    RelayConfig
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

type GRPCConfig struct {
	Port string `env:"GRPC_PORT" env-default:":50053"`
}

type PostgresConfig struct {
	URL string `env:"DATABASE_URL" env-required:"true"`
}

type RedisConfig struct {
	Host     string        `env:"REDIS_HOST" env-default:"localhost"`
	Port     string        `env:"REDIS_PORT" env-default:"6379"`
	Password string        `env:"REDIS_PASSWORD" env-default:""`
	DB       int           `env:"REDIS_DB" env-default:"1"`
	FXTTL    time.Duration `env:"REDIS_FX_TTL" env-default:"1h"`
}

type RabbitMQConfig struct {
	URL string `env:"RABBITMQ_URL" env-default:"amqp://guest:guest@localhost:5672"`
}

type FXConfig struct {
	DefaultCurrency domain.Currency `env:"FX_DEFAULT_CURRENCY" env-default:"USD"`
}

type BillingConfig struct {
	PlatformFeePercent int `env:"PLATFORM_FEE_PERCENT" env-default:"15"`
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
