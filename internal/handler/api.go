package handler

import (
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

func (h *APIHandler) handleSend(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		Message string `json:"message"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON body", http.StatusBadRequest)
		return
	}
	if req.Message == "" {
		http.Error(w, "message cannot be empty", http.StatusBadRequest)
		return
	}

	if err := h.SQS.Send(r.Context(), req.Message); err != nil {
		h.Log.Error("failed to send message", "err", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	jsonResponse(w, map[string]string{"status": "ok", "message": "message sent successfully"})
}

func (h *APIHandler) handleMessages(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	msgs, err := h.SQS.Receive(r.Context(), 10)
	if err != nil {
		h.Log.Error("failed to receive messages", "err", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	jsonResponse(w, msgs)
}

func (h *APIHandler) handlePurge(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if err := h.SQS.Purge(r.Context()); err != nil {
		h.Log.Error("failed to purge queue", "err", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	jsonResponse(w, map[string]string{"status": "ok", "message": "queue purged successfully"})
}

func (h *APIHandler) handleInfo(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	info := h.SQS.Info(r.Context())
	jsonResponse(w, info)
}

func jsonResponse(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(data)
}
