package config

import (
	"fmt"
	"os"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
)

type Config struct {
	Env      string `env:"ENV" env-default:"local"`
	HTTP     HTTPConfig
	JWT      JWTConfig
	Services ServicesConfig
}

type HTTPConfig struct {
	Port         string        `env:"HTTP_PORT" env-default:":8080"`
	ReadTimeout  time.Duration `env:"HTTP_READ_TIMEOUT" env-default:"5s"`
	WriteTimeout time.Duration `env:"HTTP_WRITE_TIMEOUT" env-default:"10s"`
	IdleTimeout  time.Duration `env:"HTTP_IDLE_TIMEOUT" env-default:"60s"`
}

type JWTConfig struct {
	SecretKey string `env:"JWT_SECRET" env-required:"true"`
}

type ServicesConfig struct {
	User    UserServiceConfig
	Catalog CatalogServiceConfig
}

type UserServiceConfig struct {
	Addr    string        `env:"USER_SERVICE_GRPC_ADDR" env-default:"localhost:50051"`
	Timeout time.Duration `env:"USER_SERVICE_TIMEOUT" env-default:"5s"`
}

type CatalogServiceConfig struct {
	Addr    string        `env:"CATALOG_SERVICE_GRPC_ADDR" env-default:"localhost:50052"`
	Timeout time.Duration `env:"CATALOG_SERVICE_TIMEOUT" env-default:"5s"`
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
