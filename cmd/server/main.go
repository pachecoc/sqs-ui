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
	"github.com/pachecoc/sqs-ui/internal/settings"
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
	appCfg := settings.Load(baseLog)

	// Rebuild logger with configured level
	log := logging.NewLogger(appCfg.LogLevel)

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
		appCfg.QueueName,
		appCfg.QueueURL,
		log,
	)

	// Print appCfg object
	// log.Info("configuration", "config", svc)
	// os.Exit(0)

	// HTTP API
	api := handler.NewAPIHandler(svc, log)
	mux := http.NewServeMux()
	api.RegisterRoutes(mux)
	mux.Handle("/", http.FileServer(http.Dir("./web")))

	// HTTP server with timeouts
	server := &http.Server{
		Addr:         ":" + appCfg.Port,
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	// Start server
	go func() {
		log.Info("starting server", "port", appCfg.Port)
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
		os.Exit(1)
	} else {
		log.Info("shutdown complete")
	}
}
