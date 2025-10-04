package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	appconfig "github.com/pachecoc/sqs-ui/internal/config"
	"github.com/pachecoc/sqs-ui/internal/handler"
	"github.com/pachecoc/sqs-ui/internal/logging"
	"github.com/pachecoc/sqs-ui/internal/service"
)

func main() {
	// Context cancelled on SIGINT/SIGTERM
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// Base logger (info) for early config errors
	baseLog := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	// Load config from env
	cfg := appconfig.Load(baseLog)

	// Rebuild logger with configured level
	log := logging.NewLogger(cfg.LogLevel)

	// Load AWS default config
	awsCfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		log.Error("failed to load AWS config", "error", err)
		os.Exit(1)
	}

	// Construct SQS client
	sqsClient := sqs.NewFromConfig(awsCfg)

	// Service with configured timeouts and receive behavior
	svc := service.NewSQSService(
		ctx,
		sqsClient,
		cfg.QueueName,
		cfg.QueueURL,
		log,
		cfg.DefaultMessageGroupID,
		cfg.RequestTimeout,
		cfg.ReceiveMaxMessages,
		cfg.ReceiveWaitSeconds,
		cfg.ReceiveVisibilityTimeout,
		cfg.ReceiveSingleCall,
	)

	// HTTP API
	api := handler.NewAPIHandler(svc, log)
	mux := http.NewServeMux()
	api.RegisterRoutes(mux)
	mux.Handle("/", http.FileServer(http.Dir("./web")))

	// HTTP server with sane timeouts
	server := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
	}

	// Start server
	go func() {
		log.Info("starting server", "port", cfg.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	// Wait for signal
	<-ctx.Done()
	log.Info("shutting down gracefully", "timeout_seconds", 3)

	// Graceful shutdown
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Error("graceful shutdown failed", "error", err)
	} else {
		log.Info("shutdown complete")
	}
}
