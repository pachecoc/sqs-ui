package main

import (
	"context"
	"html/template"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/pachecoc/sqs-ui/internal/awsclient"
	"github.com/pachecoc/sqs-ui/internal/config"
	"github.com/pachecoc/sqs-ui/internal/handler"
	"github.com/pachecoc/sqs-ui/internal/service"
)

func main() {
	// --- JSON structured logger ---
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	logger.Info("🚀 Starting SQS UI Server...")

	// --- Load configuration ---
	cfg := config.Load()

	// --- AWS SQS client ---
	ctx := context.Background()
	sqsClient := awsclient.NewSQSClient(ctx, cfg.QueueURL)

	// --- SQS service ---
	sqsSvc := service.NewSQSService(ctx, sqsClient, cfg.QueueName, cfg.QueueURL, logger)

	// --- Handlers ---
	apiHandler := handler.NewAPIHandler(sqsSvc, logger)

	tmpl, err := template.ParseFiles("internal/templates/index.html")
	if err != nil {
		logger.Error("failed to load HTML template", "error", err)
		os.Exit(1)
	}
	uiHandler := handler.NewUIHandler(tmpl, logger)

	mux := http.NewServeMux()
	apiHandler.RegisterRoutes(mux)
	uiHandler.RegisterRoutes(mux)

	port := cfg.Port
	if port == "" {
		port = "8080"
	}
	addr := ":" + port

	server := &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	// --- Graceful shutdown setup ---
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		logger.Info("🌐 Server listening", "address", addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("server failed", "error", err)
			os.Exit(1)
		}
	}()

	// Wait for termination
	<-stop
	logger.Info("🛑 Shutdown signal received, stopping server...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Error("❌ Server forced to shutdown", "error", err)
	} else {
		logger.Info("✅ Server stopped gracefully")
	}
}
