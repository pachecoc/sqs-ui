package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"sqs-demo/internal/awsclient"
	"sqs-demo/internal/config"
	"sqs-demo/internal/handler"
	"sqs-demo/internal/logging"
	"sqs-demo/internal/service"
	"syscall"
	"time"
)

func main() {
	ctx := context.Background()
	log := logging.NewLogger()

	cfg := config.Load()
	client := awsclient.NewSQSClient(ctx)
	sqsSvc := service.NewSQSService(ctx, client, cfg.QueueName, cfg.QueueURL, log)
	api := handler.NewAPIHandler(sqsSvc, log)

	mux := http.NewServeMux()
	mux.HandleFunc("/", handler.ServeUI)
	mux.HandleFunc("/send", api.SendMessage)
	mux.HandleFunc("/messages", api.GetMessages)
	mux.HandleFunc("/info", api.Info)

	srv := &http.Server{Addr: ":" + cfg.Port, Handler: mux}

	go func() {
		log.Info("server starting", "port", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("server error", "err", err)
			os.Exit(1)
		}
	}()

	// graceful shutdown
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop
	log.Info("shutting down server...")

	ctxTimeout, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	_ = srv.Shutdown(ctxTimeout)
	log.Info("server stopped gracefully")
}
