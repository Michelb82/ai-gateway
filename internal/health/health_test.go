package health_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/buildright/construction-ai-gateway/internal/capability"
	"github.com/buildright/construction-ai-gateway/internal/health"
)

type fakeModels struct {
	available map[string]bool
	err       error
}

func (f *fakeModels) ModelAvailable(ctx context.Context, name string) (bool, error) {
	if f.err != nil {
		return false, f.err
	}
	return f.available[name], nil
}

func TestHealthJSONReady(t *testing.T) {
	reg := capability.NewRegistry("qwen3:1.7b", "qwen3:4b")
	models := &fakeModels{available: map[string]bool{
		"qwen3:1.7b": true,
		"qwen3:4b":   true,
	}}
	handler := health.NewHandler(reg, models)
	mux := http.NewServeMux()
	handler.Register(mux)

	req := httptest.NewRequest(http.MethodGet, "/health.json", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}

	var report health.Report
	if err := json.Unmarshal(rec.Body.Bytes(), &report); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if report.Status != health.StatusReady {
		t.Fatalf("status = %q", report.Status)
	}
	if len(report.Capabilities) != 2 {
		t.Fatalf("capabilities = %d", len(report.Capabilities))
	}
}

func TestHealthJSONNotReady(t *testing.T) {
	reg := capability.NewRegistry("qwen3:1.7b", "qwen3:4b")
	models := &fakeModels{available: map[string]bool{
		"qwen3:1.7b": true,
		"qwen3:4b":   false,
	}}
	handler := health.NewHandler(reg, models)
	mux := http.NewServeMux()
	handler.Register(mux)

	req := httptest.NewRequest(http.MethodGet, "/health.json", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d", rec.Code)
	}

	var report health.Report
	_ = json.Unmarshal(rec.Body.Bytes(), &report)
	if report.Status != health.StatusNotReady {
		t.Fatalf("status = %q", report.Status)
	}
	found := false
	for _, item := range report.Capabilities {
		if item.Capability == capability.IntentClassification {
			found = true
			if item.Status != health.StatusUnavailable {
				t.Fatalf("intent status = %q", item.Status)
			}
			if item.Error == "" {
				t.Fatalf("expected error")
			}
		}
	}
	if !found {
		t.Fatalf("intent capability missing")
	}
}

func TestHealthHTML(t *testing.T) {
	reg := capability.NewRegistry("qwen3:1.7b", "qwen3:4b")
	models := &fakeModels{available: map[string]bool{
		"qwen3:1.7b": true,
		"qwen3:4b":   false,
	}}
	handler := health.NewHandler(reg, models)
	mux := http.NewServeMux()
	handler.Register(mux)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, `id="routing"`) {
		t.Fatalf("missing routing div")
	}
	if !strings.Contains(body, `id="intent-classification"`) {
		t.Fatalf("missing intent div")
	}
	if !strings.Contains(body, "unavailable") {
		t.Fatalf("missing unavailable status")
	}
	if !strings.Contains(body, `style="color: red"`) {
		t.Fatalf("missing red color")
	}
}
