package config

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	. "blum-test/common/logger"

	"github.com/joho/godotenv"
	"github.com/kelseyhightower/envconfig"
)

type AppConfig struct {
	Name     string   `envconfig:"APP_NAME" required:"true"`
	LogLevel string   `envconfig:"LOG_LEVEL" default:"WARN"`
	Service  *Service `envconfig:"SERVICE"`

	HTTPServer *HTTPServer `envconfig:"HTTP_SERVER"`

	Postgres  *Postgres  `envconfig:"POSTGRES_DB"`
	FastForex *FastForex `envconfig:"FAST_FOREX"`
}

type Service struct {
	RatePollingInterval     time.Duration `envconfig:"RATE_POLLING_INTERVAL" default:"60s"`
	CurrencyPollingInterval time.Duration `envconfig:"CURRENCY_POLLING_INTERVAL" default:"5s"`
}

type FastForex struct {
	ApiKey         string        `envconfig:"API_KEY" required:"true"`
	RequestTimeout time.Duration `envconfig:"REQUEST_TIMEOUT" default:"30s"`
	RetriesCount   int           `envconfig:"RETRIES_COUNT" default:"3"`
}

func getEnvFilenames() []string {
	return []string{".env.local", ".env"}
}

func LoadConfig(ctx context.Context) (*AppConfig, error) {
	for _, fileName := range getEnvFilenames() {
		err := godotenv.Load(fileName)
		if err != nil {
			JSONLogger.Error("error loading env file", slog.String("filename", fileName), err)
		}
	}

	var cfg AppConfig
	if err := envconfig.Process("", &cfg); err != nil {
		JSONLogger.Error("cannot process envs", err)
		return nil, fmt.Errorf("cannot process envs: %w", err)
	} else {
		JSONLogger.Info("Config initialized")
	}

	return &cfg, nil
}
