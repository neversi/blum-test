package logger

import (
	"log/slog"
	"os"
	"sync"
)

var JSONLogger *slog.Logger = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))

var onceInit sync.Once

func InitLogger(appName string, level string) error {
	var slevel slog.Level
	if err := slevel.UnmarshalText([]byte(level)); err != nil {
		return err
	}

	JSONLogger = slog.New(slog.NewJSONHandler(
		os.Stdout,
		&slog.HandlerOptions{
			Level: slevel,
		},
	),
	).
		With(slog.Any("app", appName))

	return nil
}
