// Package logging builds the slog logger shared by all services.
package logging

import (
	"log/slog"
	"os"
	"strings"
)

// New returns a stdout logger honoring LOG_LEVEL (debug|info|warn|error,
// default info) and LOG_FORMAT (text|json, default text). Unknown values
// fall back to the defaults rather than failing startup.
func New() *slog.Logger {
	opts := &slog.HandlerOptions{Level: level()}
	if strings.EqualFold(os.Getenv("LOG_FORMAT"), "json") {
		return slog.New(slog.NewJSONHandler(os.Stdout, opts))
	}
	return slog.New(slog.NewTextHandler(os.Stdout, opts))
}

func level() slog.Level {
	switch strings.ToLower(os.Getenv("LOG_LEVEL")) {
	case "debug":
		return slog.LevelDebug
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
