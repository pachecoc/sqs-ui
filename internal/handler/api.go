package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sqs-demo/internal/service"

	"log/slog"
)

type APIHandler struct {
	SQS *service.SQSService
	Log *slog.Logger
}

func NewAPIHandler(sqs *service.SQSService, log *slog.Logger) *APIHandler {
	return &APIHandler{SQS: sqs, Log: log}
}

func (h *APIHandler) SendMessage(w http.ResponseWriter, r *http.Request) {
	msg := r.URL.Query().Get("msg")
	if msg == "" {
		msg = "Hello from SQS Demo!"
	}
	if err := h.SQS.Send(context.TODO(), msg); err != nil {
		h.Log.Error("failed to send message", "err", err)
		http.Error(w, fmt.Sprintf("Failed to send: %v", err), 500)
		return
	}
	fmt.Fprintf(w, "✅ Message sent: %s", msg)
}

func (h *APIHandler) GetMessages(w http.ResponseWriter, r *http.Request) {
	msgs, err := h.SQS.Receive(context.TODO(), 5)
	if err != nil {
		h.Log.Error("failed to get messages", "err", err)
		http.Error(w, fmt.Sprintf("Error: %v", err), 500)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(msgs)
}

func (h *APIHandler) Info(w http.ResponseWriter, r *http.Request) {
	info := map[string]string{"queueURL": h.SQS.QueueURL, "status": "ok"}
	json.NewEncoder(w).Encode(info)
}
