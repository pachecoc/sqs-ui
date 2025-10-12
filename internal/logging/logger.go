package logging

import (
	"log/slog"
	"os"
	"strings"
)

// NewLogger builds a slog JSON logger honoring LOG_LEVEL (debug|info|warn|error).
func NewLogger(levelStr string) *slog.Logger {
	level := slog.LevelInfo
	switch strings.ToLower(strings.TrimSpace(levelStr)) {
	case "debug":
		level = slog.LevelDebug
	case "warn", "warning":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	}
	// Using JSON handler for structured output.
	return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level}))
}
