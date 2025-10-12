package handler

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sqs"

	"github.com/pachecoc/sqs-ui/internal/service"
	"github.com/pachecoc/sqs-ui/internal/version"
)

// APIHandler provides HTTP endpoints for interacting with SQS.
type APIHandler struct {
	SQS *service.SQSService
	Log *slog.Logger
	mu  sync.RWMutex // switched to RWMutex: reads dominate, queue change is rare
}

// NewAPIHandler creates a new APIHandler.
func NewAPIHandler(sqs *service.SQSService, log *slog.Logger) *APIHandler {
	return &APIHandler{SQS: sqs, Log: log}
}

// requireQueue ensures a queue name or URL is configured before executing the handler.
func (h *APIHandler) requireQueue(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		svc := h.getService()
		if svc == nil {
			respondError(w, http.StatusServiceUnavailable, errors.New("service unavailable"))
			return
		}
		if err := svc.EnsureQueueConfigured(); err != nil {
			respondError(w, http.StatusBadRequest, err)
			return
		}
		next(w, r)
	}
}

// RegisterRoutes wires all HTTP endpoints.
func (h *APIHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/send", h.requireQueue(h.handleSend))
	mux.HandleFunc("/api/messages", h.requireQueue(h.handleMessages))
	mux.HandleFunc("/api/purge", h.requireQueue(h.handlePurge))

	// Queue can be (re)configured at runtime
	mux.HandleFunc("/api/config/queue", h.handleChangeQueue)

	// Informational endpoints
	mux.HandleFunc("/info", h.handleInfo)
	mux.HandleFunc("/healthz", h.handleHealth)
}

// handleSend accepts JSON { "message": "<text>" } and forwards to SQS.
func (h *APIHandler) handleSend(w http.ResponseWriter, r *http.Request) {
	if !enforceMethod(w, r, http.MethodPost) {
		return
	}
	if ct := r.Header.Get("Content-Type"); ct != "" && ct != "application/json" {
		respondError(w, http.StatusUnsupportedMediaType, errors.New("Content-Type must be application/json"))
		return
	}

	var req struct {
		Message string `json:"message"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, err)
		return
	}
	if req.Message == "" {
		respondError(w, http.StatusBadRequest, errors.New("message cannot be empty"))
		return
	}

	svc := h.getService()
	if svc == nil {
		respondError(w, http.StatusServiceUnavailable, errors.New("service unavailable"))
		return
	}

	if err := svc.Send(r.Context(), req.Message); err != nil {
		h.Log.Error("failed to send message", "error", err)
		respondError(w, http.StatusInternalServerError, err)
		return
	}

	respondJSON(w, http.StatusOK, map[string]any{
		"status":  "ok",
		"message": "message sent successfully",
	})
}

// handleMessages fetches available messages (non-destructive peek).
func (h *APIHandler) handleMessages(w http.ResponseWriter, r *http.Request) {
	if !enforceMethod(w, r, http.MethodGet) {
		return
	}

	svc := h.getService()
	if svc == nil {
		respondError(w, http.StatusServiceUnavailable, errors.New("service unavailable"))
		return
	}

	msgs, err := svc.Fetch(r.Context(), 0)
	if err != nil {
		h.Log.Error("failed to receive messages", "error", err)
		respondError(w, http.StatusInternalServerError, err)
		return
	}
	respondJSON(w, http.StatusOK, msgs)
}

// handlePurge deletes all messages presently in the queue.
func (h *APIHandler) handlePurge(w http.ResponseWriter, r *http.Request) {
	if !enforceMethod(w, r, http.MethodPost) {
		return
	}

	svc := h.getService()
	if svc == nil {
		respondError(w, http.StatusServiceUnavailable, errors.New("service unavailable"))
		return
	}

	if err := svc.Purge(r.Context()); err != nil {
		h.Log.Error("failed to purge queue", "error", err)
		respondError(w, http.StatusInternalServerError, err)
		return
	}
	respondJSON(w, http.StatusOK, map[string]string{
		"status":  "ok",
		"message": "queue purged successfully",
	})
}

// handleInfo returns summary queue metrics (never errors HTTP-level unless internal encoding fails).
func (h *APIHandler) handleInfo(w http.ResponseWriter, r *http.Request) {
	if !enforceMethod(w, r, http.MethodGet) {
		return
	}
	svc := h.getService()
	// Even if nil, return a not_connected semantics
	if svc == nil {
		respondJSON(w, http.StatusOK, map[string]any{
			"status":  "not_connected",
			"error":   "service unavailable",
			"message": "no SQS service configured",
		})
		return
	}
	info := svc.Info(r.Context())
	respondJSON(w, http.StatusOK, info)
}

// handleChangeQueue updates the SQS queue at runtime.
func (h *APIHandler) handleChangeQueue(w http.ResponseWriter, r *http.Request) {
	if !enforceMethod(w, r, http.MethodPost) {
		return
	}
	if ct := r.Header.Get("Content-Type"); ct != "" && ct != "application/json" {
		respondError(w, http.StatusUnsupportedMediaType, errors.New("Content-Type must be application/json"))
		return
	}

	var body struct {
		QueueName string `json:"queue_name"`
		QueueURL  string `json:"queue_url"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		respondError(w, http.StatusBadRequest, err)
		return
	}
	if body.QueueName == "" && body.QueueURL == "" {
		respondError(w, http.StatusBadRequest, errors.New("queue_name or queue_url must be provided"))
		return
	}

	// Short timeout to avoid long hangs on AWS metadata/STS
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	awsCfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		h.Log.Warn("failed to reload AWS config", "error", err)
		respondError(w, http.StatusServiceUnavailable, errors.New("could not reload AWS config"))
		return
	}

	client := sqs.NewFromConfig(awsCfg)
	newSvc := service.NewSQSService(ctx, client, body.QueueName, body.QueueURL, awsCfg.Region, h.Log)

	h.mu.Lock()
	h.SQS = newSvc
	h.mu.Unlock()

	h.Log.Info("SQS queue updated", "queue_name", newSvc.QueueName, "queue_url", newSvc.QueueURL)

	respondJSON(w, http.StatusOK, map[string]any{
		"status":      "ok",
		"queue_name":  newSvc.QueueName,
		"queue_url":   newSvc.QueueURL,
		"reconnected": newSvc.QueueURL != "",
	})
}

// handleHealth returns a simple liveness probe and version info.
func (h *APIHandler) handleHealth(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, http.StatusOK, map[string]string{
		"status":     "ok",
		"version":    version.Version,
		"commit":     version.Commit,
		"build_time": version.BuildTime,
	})
}

/*
Helper functions
*/

func (h *APIHandler) getService() *service.SQSService {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.SQS
}

// enforceMethod ensures the request verb matches and sets Allow header on mismatch.
func enforceMethod(w http.ResponseWriter, r *http.Request, allowed string) bool {
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return false
	}
	if r.Method != allowed {
		w.Header().Set("Allow", allowed)
		respondError(w, http.StatusMethodNotAllowed, errors.New("method not allowed"))
		return false
	}
	return true
}

func respondJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func respondError(w http.ResponseWriter, status int, err error) {
	payload := map[string]string{
		"error":  http.StatusText(status),
	}
	if err != nil {
		payload["detail"] = err.Error()
	}
	respondJSON(w, status, payload)
}
