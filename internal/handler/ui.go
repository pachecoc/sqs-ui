package handler

import (
	"html/template"
	"log/slog"
	"net/http"
	"strings"
)

type UIHandler struct {
	Tmpl *template.Template
	Log  *slog.Logger
}

func NewUIHandler(tmpl *template.Template, log *slog.Logger) *UIHandler {
	return &UIHandler{Tmpl: tmpl, Log: log}
}

func (h *UIHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Prevent serving HTML for API routes
		if strings.HasPrefix(r.URL.Path, "/api/") {
			http.NotFound(w, r)
			return
		}
		h.serveUI(w, r)
	})
}

func (h *UIHandler) serveUI(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	h.Log.Info("🌐 Serving UI", "path", r.URL.Path, "method", r.Method)
	if err := h.Tmpl.Execute(w, nil); err != nil {
		h.Log.Error("❌ Failed to render UI template", "error", err)
		http.Error(w, "failed to render UI", http.StatusInternalServerError)
	}
}
