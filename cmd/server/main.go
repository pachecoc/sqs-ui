package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sqs"

	"github.com/pachecoc/sqs-ui/internal/handler"
	"github.com/pachecoc/sqs-ui/internal/logging"
	"github.com/pachecoc/sqs-ui/internal/service"
	"github.com/pachecoc/sqs-ui/internal/settings"
	"github.com/pachecoc/sqs-ui/internal/version"
)

func main() {
	if handleVersionFlag() {
		return
	}

	// Context canceled on SIGINT/SIGTERM
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// Early logger (info JSON)
	baseLog := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	// Load configuration
	appCfg := settings.Load(baseLog)

	// Rebuild logger using configured level
	log := logging.NewLogger(appCfg.LogLevel)

	// Load AWS config (best effort)
	awsCfg, awsErr := config.LoadDefaultConfig(ctx)
	if awsErr != nil {
		log.Warn("could not load AWS config", "error", awsErr)
	} else if awsCfg.Region != "" {
		log.Info("aws region detected", "region", awsCfg.Region)
	} else {
		log.Info("aws region not set")
	}

	var sqsClient *sqs.Client
	if awsErr == nil {
		sqsClient = sqs.NewFromConfig(awsCfg)
	}

	// Build SQS service (idle mode if no queue config)
	svc := buildSQSService(ctx, sqsClient, awsCfg.Region, appCfg.QueueName, appCfg.QueueURL, log)

	// HTTP routing
	mux := http.NewServeMux()
	api := handler.NewAPIHandler(svc, log)
	api.RegisterRoutes(mux)
	mux.Handle("/", http.FileServer(http.Dir("./web")))

	server := &http.Server{
		Addr:         ":" + appCfg.Port,
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server
	go func() {
		log.Info("starting server", "port", appCfg.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	// Wait for termination
	<-ctx.Done()
	log.Info("shutting down", "timeout_seconds", 3)

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Error("graceful shutdown failed", "error", err)
		os.Exit(1)
	}

	log.Info("shutdown complete")
}

func handleVersionFlag() bool {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "--version", "-v":
			fmt.Printf("Version: %s\nCommit: %s\nBuilt: %s\n",
				version.Version, version.Commit, version.BuildTime)
			return true
		}
	}
	return false
}

func buildSQSService(
	ctx context.Context,
	client *sqs.Client,
	region string,
	queueName string,
	queueURL string,
	log *slog.Logger,
) *service.SQSService {
	if queueName == "" && queueURL == "" {
		log.Warn("no queue name or URL configured - running in idle mode")
		return &service.SQSService{
			Client:    client,
			QueueName: "",
			QueueURL:  "",
			Region:    region,
			Log:       log,
		}
	}
	return service.NewSQSService(ctx, client, queueName, queueURL, region, log)
}
