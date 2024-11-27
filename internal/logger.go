package internal

import (
	"log/slog"
	"os"
)

var Logger *slog.Logger

func LoadLogger() {
	opts := slog.HandlerOptions{}

	if os.Getenv("DEBUG") == "true" {
		opts.Level = slog.LevelDebug
	}

	Logger = slog.New(slog.NewTextHandler(os.Stdout, &opts))
	// Logger = slog.New(slog.NewJSONHandler(os.Stdout, &opts))
}
