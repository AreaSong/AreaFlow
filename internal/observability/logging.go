package observability

import (
	"log/slog"
	"os"
)

func ConfigureJSONLogging() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})))
}
