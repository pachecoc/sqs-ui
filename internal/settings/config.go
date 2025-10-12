package settings

import (
	"log/slog"
	"os"
	"strconv"
	"strings"
)

// AppConfig holds application runtime parameters (populated from environment).
type AppConfig struct {
	QueueName              string
	QueueURL               string
	LogLevel               string
	Port                   string
}

// Load reads environment variables, applying defaults and validation.
func Load(log *slog.Logger) AppConfig {

	// Read environment variables
	queueName := os.Getenv("QUEUE_NAME")
	queueURL := os.Getenv("QUEUE_URL")
	port := os.Getenv("PORT")
	logLevel := os.Getenv("LOG_LEVEL")

	// Default port
	if port == "" {
		port = "8080"
	}

	// Default log level
	if logLevel == "" {
		logLevel = "info"
	}

	return AppConfig{
		QueueName:              queueName,
		QueueURL:               queueURL,
		LogLevel:               logLevel,
		Port:                   port,
	}
}

func parseIntEnv(k string, def int) int {
	// Safe integer parser with fallback
	v := os.Getenv(k)
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil || n <= 0 {
		return def
	}
	return n
}

func parseBoolEnv(k string, def bool) bool {
	v := strings.ToLower(os.Getenv(k))
	if v == "" {
		return def
	}
	switch v {
	case "1", "true", "yes", "y":
		return true
	case "0", "false", "no", "n":
		return false
	default:
		return def
	}
}
