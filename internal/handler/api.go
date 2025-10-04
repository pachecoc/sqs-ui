package handler

import (
	"encoding/json"
	"net/http"

	"github.com/pachecoc/sqs-ui/internal/service"
	"log/slog"
)

// APIHandler provides HTTP endpoints for interacting with SQS.
type APIHandler struct {
	SQS *service.SQSService
	Log *slog.Logger
}

// NewAPIHandler creates a new APIHandler.
func NewAPIHandler(sqs *service.SQSService, log *slog.Logger) *APIHandler {
	return &APIHandler{SQS: sqs, Log: log}
}

// RegisterRoutes wires all HTTP endpoints.
func (h *APIHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/send", h.handleSend)
	mux.HandleFunc("/api/messages", h.handleMessages)
	mux.HandleFunc("/api/purge", h.handlePurge)
	mux.HandleFunc("/info", h.handleInfo)
	mux.HandleFunc("/healthz", h.handleHealth)
}

// handleSend accepts a JSON body { "message": "<text>" } and forwards to SQS.
func (h *APIHandler) handleSend(w http.ResponseWriter, r *http.Request) {
	if !method(w, r, http.MethodPost) {
		return
	}
	var req struct {
		Message string `json:"message"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if req.Message == "" {
		http.Error(w, "message cannot be empty", http.StatusBadRequest)
		return
	}
	if err := h.SQS.Send(r.Context(), req.Message); err != nil {
		h.Log.Error("failed to send message", "err", err)
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	info := h.SQS.Info(r.Context())
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"status":  "ok",
		"message": "message sent successfully",
		"info":    info,
	})
}

// handleMessages fetches available messages (non-destructive peek).
func (h *APIHandler) handleMessages(w http.ResponseWriter, r *http.Request) {
	if !method(w, r, http.MethodGet) {
		return
	}
	// Always aggregate all immediately available messages now.
	msgs, err := h.SQS.ReceiveAll(r.Context(), 0)
	if err != nil {
		h.Log.Error("failed to receive messages", "err", err)
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, msgs)
}

// handlePurge deletes all messages presently in the queue.
func (h *APIHandler) handlePurge(w http.ResponseWriter, r *http.Request) {
	if !method(w, r, http.MethodPost) {
		return
	}
	if err := h.SQS.Purge(r.Context()); err != nil {
		h.Log.Error("failed to purge queue", "err", err)
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "message": "queue purged successfully"})
}

// handleInfo returns summary queue metrics.
func (h *APIHandler) handleInfo(w http.ResponseWriter, r *http.Request) {
	if !method(w, r, http.MethodGet) {
		return
	}
	info := h.SQS.Info(r.Context())
	writeJSON(w, http.StatusOK, info)
}

// handleHealth simple liveness probe.
func (h *APIHandler) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// Helpers

// method enforces allowed method and handles OPTIONS early return.
func method(w http.ResponseWriter, r *http.Request, allowed string) bool {
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return false
	}
	if r.Method != allowed {
		writeError(w, http.StatusMethodNotAllowed, nil)
		return false
	}
	return true
}

// writeJSON sends JSON with status and headers set.
func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// writeError wraps an error into a JSON error envelope.
func writeError(w http.ResponseWriter, status int, err error) {
	resp := map[string]string{"error": http.StatusText(status)}
	if err != nil {
		resp["detail"] = err.Error()
	}
	writeJSON(w, status, resp)
}
