package config

import (
	"os"
	"log/slog"
)

type AppConfig struct {
	QueueName string
	QueueURL  string
	Port      string
}

func Load(log *slog.Logger) AppConfig {
	queueName := os.Getenv("QUEUE_NAME")
	queueURL := os.Getenv("QUEUE_URL")
	port := os.Getenv("PORT")

	if port == "" {
		port = "8080"
	}

	if queueName == "" && queueURL == "" {
		log.Error("missing QUEUE_NAME or QUEUE_URL environment variable")
		os.Exit(1)
	}

	return AppConfig{
		QueueName: queueName,
		QueueURL:  queueURL,
		Port:      port,
	}
}
