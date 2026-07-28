package health

import (
	"context"
	"encoding/json"
	"fmt"
	"html"
	"net/http"
	"strings"
	"time"

	"github.com/buildright/construction-ai-gateway/internal/capability"
)

const (
	StatusReady       = "ready"
	StatusNotReady    = "not_ready"
	StatusAvailable   = "available"
	StatusUnavailable = "unavailable"
)

type ModelChecker interface {
	ModelAvailable(ctx context.Context, name string) (bool, error)
}

type CapabilityStatus struct {
	Capability string `json:"capability"`
	Status     string `json:"status"`
	Model      string `json:"model"`
	Error      string `json:"error,omitempty"`
}

type Report struct {
	Status       string             `json:"status"`
	Capabilities []CapabilityStatus `json:"capabilities"`
}

type Handler struct {
	registry *capability.Registry
	models   ModelChecker
}

func NewHandler(registry *capability.Registry, models ModelChecker) *Handler {
	return &Handler{
		registry: registry,
		models:   models,
	}
}

func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("/health", h.handleHTML)
	mux.HandleFunc("/health.json", h.handleJSON)
}

func (h *Handler) handleHTML(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	report := h.buildReport(r.Context())
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(httpStatus(report.Status))
	_, _ = w.Write([]byte(renderHTML(report)))
}

func (h *Handler) handleJSON(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	report := h.buildReport(r.Context())
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(httpStatus(report.Status))
	_ = json.NewEncoder(w).Encode(report)
}

func (h *Handler) buildReport(ctx context.Context) Report {
	defs := h.registry.All()
	statuses := make([]CapabilityStatus, 0, len(defs))
	overall := StatusReady

	for _, def := range defs {
		status := CapabilityStatus{
			Capability: def.Name,
			Model:      def.Model,
			Status:     StatusAvailable,
		}

		available, err := h.models.ModelAvailable(ctx, def.Model)
		if err != nil {
			status.Status = StatusUnavailable
			status.Error = err.Error()
			overall = StatusNotReady
		} else if !available {
			status.Status = StatusUnavailable
			status.Error = "model not found on Ollama"
			overall = StatusNotReady
		}

		statuses = append(statuses, status)
	}

	return Report{
		Status:       overall,
		Capabilities: statuses,
	}
}

func httpStatus(overall string) int {
	if overall == StatusReady {
		return http.StatusOK
	}
	return http.StatusServiceUnavailable
}

func renderHTML(report Report) string {
	var b strings.Builder
	b.WriteString("<!DOCTYPE html>\n<html lang=\"en\">\n<head><meta charset=\"utf-8\"><title>AI capability health</title></head>\n<body>\n")
	b.WriteString("<h1>AI capability health</h1>\n")
	b.WriteString(fmt.Sprintf("<p>Overall: <span id=\"overall\" style=\"color: %s\">%s</span></p>\n",
		colorFor(report.Status), html.EscapeString(report.Status)))

	for _, item := range report.Capabilities {
		b.WriteString(fmt.Sprintf("<div id=\"%s\">%s: <span style=\"color: %s\">%s</span></div>\n",
			html.EscapeString(item.Capability),
			html.EscapeString(item.Capability),
			colorFor(item.Status),
			html.EscapeString(item.Status),
		))
	}

	b.WriteString("</body>\n</html>\n")
	return b.String()
}

func colorFor(status string) string {
	switch status {
	case StatusReady, StatusAvailable:
		return "green"
	default:
		return "red"
	}
}

func NewServer(addr string, handler *Handler) *http.Server {
	mux := http.NewServeMux()
	handler.Register(mux)
	return &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}
}
