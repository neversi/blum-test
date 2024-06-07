package config

import "time"

type HTTPServer struct {
	Host            string        `default:"0.0.0.0"`
	Port            uint16        `envconfig:"PORT" default:"8080"`
	ShutdownTimeout time.Duration `envconfig:"SHUTDOWN_TIMEOUT" default:"30s"`
}
