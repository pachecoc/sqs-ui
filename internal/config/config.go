package config

import (
	"log/slog"
	"os"
	"strconv"
	"strings"
	"time"
)

// AppConfig holds application runtime parameters (populated from environment).
type AppConfig struct {
	QueueName              string
	QueueURL               string
	Port                   string
	LogLevel               string
	DefaultMessageGroupID  string
	RequestTimeout         time.Duration
	ReceiveMaxMessages     int32
	ReceiveWaitSeconds     int32
	ReceiveVisibilityTimeout int32
	ReceiveSingleCall      bool
}

// Load reads environment variables, applying defaults and validation.
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

	logLevel := os.Getenv("LOG_LEVEL")
	if logLevel == "" {
		logLevel = "info"
	}
	msgGroup := os.Getenv("MSG_GROUP_ID")
	if msgGroup == "" {
		msgGroup = "default-group"
	}
	// REQUEST_TIMEOUT_SEC sets per-operation timeout
	timeoutSec := parseIntEnv("REQUEST_TIMEOUT_SEC", 10)
	// RECEIVE_MAX_MESSAGES caps receive batch size
	recvMax := int32(parseIntEnv("RECEIVE_MAX_MESSAGES", 10))
	waitSecs := int32(parseIntEnv("RECEIVE_WAIT_SECONDS", 5))              // SQS long poll wait (1..20)
	visibilitySecs := int32(parseIntEnv("RECEIVE_VISIBILITY_TIMEOUT", 10)) // Temporary hide after receive
	if visibilitySecs < 0 {
		visibilitySecs = 0
	}
	singleCall := parseBoolEnv("RECEIVE_SINGLE_CALL", true)

	return AppConfig{
		QueueName:              queueName,
		QueueURL:               queueURL,
		Port:                   port,
		LogLevel:               logLevel,
		DefaultMessageGroupID:  msgGroup,
		RequestTimeout:         time.Duration(timeoutSec) * time.Second,
		ReceiveMaxMessages:     recvMax,
		ReceiveWaitSeconds:     waitSecs,
		ReceiveVisibilityTimeout: visibilitySecs,
		ReceiveSingleCall:      singleCall,
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
