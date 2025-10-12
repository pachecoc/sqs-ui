package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/pachecoc/sqs-ui/internal/settings"
	"github.com/pachecoc/sqs-ui/internal/handler"
	"github.com/pachecoc/sqs-ui/internal/logging"
	"github.com/pachecoc/sqs-ui/internal/service"
	"github.com/pachecoc/sqs-ui/internal/version"
)

func main() {

	// Simple manual flag detection (before any env or config load)
	if len(os.Args) > 1 && (os.Args[1] == "--version" || os.Args[1] == "-v") {
		fmt.Printf("Version: %s\nCommit: %s\nBuilt: %s\n",
			version.Version, version.Commit, version.BuildTime)
		os.Exit(0)
	}

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
		log.Warn("could not load AWS config", "error", err)
	}

    // Ensure region is set (default to us-east-1 when not provided)
    if err == nil {
        if awsCfg.Region != "" {
            log.Info("AWS region detected", "region", awsCfg.Region)
        } else {
            log.Info("AWS region not set in credentials/config")
        }
    }

	// Create SQS client (only if config succeeded)
	var sqsClient *sqs.Client
	if err == nil {
		sqsClient = sqs.NewFromConfig(awsCfg)
	}

	// Initialize SQS service — handle empty config gracefully (idle mode).
	var svc *service.SQSService
	if appCfg.QueueName == "" && appCfg.QueueURL == "" {
		log.Warn("no QUEUE_NAME or QUEUE_URL provided — running in idle mode")
		svc = &service.SQSService{
			Client:    sqsClient,
			QueueName: "",
			QueueURL:  "",
			Region:    awsCfg.Region,
			Log:       log,
		}
	} else {
		// Function to create the SQS service and extract the queue name if URL given
		svc = service.NewSQSService(ctx, sqsClient, appCfg.QueueName, appCfg.QueueURL, awsCfg.Region, log)
	}

	// Print appCfg object
	// log.Info("configuration", "config", svc)
	// os.Exit(0)

	// Register HTTP routes.
	api := handler.NewAPIHandler(svc, log)
	mux := http.NewServeMux()
	api.RegisterRoutes(mux)
	mux.Handle("/", http.FileServer(http.Dir("./web")))

	// Configure HTTP server with sane defaults.
	server := &http.Server{
		Addr:         ":" + appCfg.Port,
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server immediately, regardless of AWS status
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
	}

	log.Info("shutdown complete")
}
