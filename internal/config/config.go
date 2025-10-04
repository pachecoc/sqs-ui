package config

import (
	"log"
	"os"
)

type Config struct {
	Port      string
	QueueURL  string
	QueueName string
}

func Load() *Config {
	port := getEnv("PORT", "8080")
	queueURL := os.Getenv("QUEUE_URL")
	queueName := os.Getenv("QUEUE_NAME")

	if queueURL == "" && queueName == "" {
		log.Fatal("Either QUEUE_URL or QUEUE_NAME must be set")
	}

	return &Config{Port: port, QueueURL: queueURL, QueueName: queueName}
}

func getEnv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
