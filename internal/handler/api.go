package handler

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/pachecoc/sqs-ui/internal/service"
	"log/slog"
)

type APIHandler struct {
	SQS *service.SQSService
	Log *slog.Logger
}

func NewAPIHandler(sqs *service.SQSService, log *slog.Logger) *APIHandler {
	return &APIHandler{SQS: sqs, Log: log}
}

func (h *APIHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/send", h.handleSend)
	mux.HandleFunc("/api/messages", h.handleMessages)
	mux.HandleFunc("/api/purge", h.handlePurge)
	mux.HandleFunc("/info", h.handleInfo)
}

// --- /api/send ---
func (h *APIHandler) handleSend(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	type sendRequest struct {
		Message string `json:"message"`
	}
	var req sendRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.Log.Error("failed to decode message body", "error", err)
		http.Error(w, "invalid JSON body", http.StatusBadRequest)
		return
	}

	if req.Message == "" {
		http.Error(w, "message cannot be empty", http.StatusBadRequest)
		return
	}

	ctx := context.Background()
	if err := h.SQS.Send(ctx, req.Message); err != nil {
		h.Log.Error("failed to send message", "err", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "ok",
		"message": "message sent successfully",
	})
}

// --- /api/messages ---
func (h *APIHandler) handleMessages(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx := context.Background()
	msgs, err := h.SQS.Receive(ctx, 10) // internally fetches all now
	if err != nil {
		h.Log.Error("failed to receive messages", "err", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if len(msgs) == 0 {
		json.NewEncoder(w).Encode([]interface{}{}) // return empty array, not null
		return
	}

	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	encoder.Encode(msgs)
}

// --- /api/purge ---
func (h *APIHandler) handlePurge(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx := context.Background()
	if err := h.SQS.Purge(ctx); err != nil {
		h.Log.Error("failed to purge queue", "err", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "ok",
		"message": "queue purged successfully",
	})
}

// --- /info ---
func (h *APIHandler) handleInfo(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	info := h.SQS.Info()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	encoder.Encode(info)
}
