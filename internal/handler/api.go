package handler

import (
	"encoding/json"
	"net/http"
	"log/slog"
	"sync"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/pachecoc/sqs-ui/internal/service"
	"github.com/pachecoc/sqs-ui/internal/version"
)

// APIHandler provides HTTP endpoints for interacting with SQS.
type APIHandler struct {
	SQS *service.SQSService
	Log *slog.Logger
	mu  sync.Mutex // protects SQS for runtime replacement
}

// NewAPIHandler creates a new APIHandler.
func NewAPIHandler(sqs *service.SQSService, log *slog.Logger) *APIHandler {
	return &APIHandler{SQS: sqs, Log: log}
}

// requireQueue is a small middleware that ensures a queue name or URL is configured.
// If not configured, it returns a 400 JSON error and does not call the next handler.
func (h *APIHandler) requireQueue(next http.HandlerFunc) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        h.mu.Lock()
        svc := h.SQS
        h.mu.Unlock()

        if svc == nil {
            writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "service unavailable"})
            return
        }
        if err := svc.EnsureQueueConfigured(); err != nil {
            writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
            return
        }
        next(w, r)
    }
}

// RegisterRoutes wires all HTTP endpoints.
func (h *APIHandler) RegisterRoutes(mux *http.ServeMux) {
    // Wrap queue-dependent endpoints with requireQueue middleware
    mux.HandleFunc("/api/send", h.requireQueue(h.handleSend))
    mux.HandleFunc("/api/messages", h.requireQueue(h.handleMessages))
    mux.HandleFunc("/api/purge", h.requireQueue(h.handlePurge))

    // endpoints that do not require a configured queue
    mux.HandleFunc("/api/config/queue", h.handleChangeQueue)
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

	h.mu.Lock()
	svc := h.SQS
	h.mu.Unlock()

	if err := svc.Send(r.Context(), req.Message); err != nil {
		h.Log.Error("failed to send message", "error", err)
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"status":  "ok",
		"message": "message sent successfully",
	})
}

// handleMessages fetches available messages (non-destructive peek).
func (h *APIHandler) handleMessages(w http.ResponseWriter, r *http.Request) {
	if !method(w, r, http.MethodGet) {
		return
	}

	h.mu.Lock()
	svc := h.SQS
	h.mu.Unlock()

	msgs, err := svc.ReceiveAll(r.Context(), 0)
	if err != nil {
		h.Log.Error("failed to receive messages", "error", err)
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

	h.mu.Lock()
	svc := h.SQS
	h.mu.Unlock()

	if err := svc.Purge(r.Context()); err != nil {
		h.Log.Error("failed to purge queue", "error", err)
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

	h.mu.Lock()
	svc := h.SQS
	h.mu.Unlock()

	info := svc.Info(r.Context())
	writeJSON(w, http.StatusOK, info)
}

// handleChangeQueue updates the SQS queue at runtime.
func (h *APIHandler) handleChangeQueue(w http.ResponseWriter, r *http.Request) {
	if !method(w, r, http.MethodPost) {
		return
	}

	var body struct {
		QueueName string `json:"queue_name"`
		QueueURL  string `json:"queue_url"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if body.QueueName == "" && body.QueueURL == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "queue_name or queue_url must be provided"})
		return
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	ctx := r.Context()
	awsCfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		h.Log.Warn("failed to reload AWS config", "error", err)
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "could not reload AWS config"})
		return
	}

	client := sqs.NewFromConfig(awsCfg)
	newSvc := service.NewSQSService(ctx, client, body.QueueName, body.QueueURL, awsCfg.Region, h.Log)
	h.SQS = newSvc

	h.Log.Info("SQS queue updated at runtime", "queue_name", newSvc.QueueName, "queue_url", newSvc.QueueURL)

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"status":      "ok",
		"queue_name":  newSvc.QueueName,
		"queue_url":   newSvc.QueueURL,
		"reconnected": newSvc.QueueURL != "",
	})
}

// handleHealth returns a simple liveness probe and version info.
func (h *APIHandler) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{
		"status":     "ok",
		"version":    version.Version,
		"commit":     version.Commit,
		"build_time": version.BuildTime,
	})
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
