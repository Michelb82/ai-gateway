package ollama_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mywebsite/construction-ai-gateway/internal/ollama"
)

func TestPoolEnsureModelsAcrossURLs(t *testing.T) {
	present := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/tags" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"models":[{"name":"present:latest"}]}`))
	}))
	defer present.Close()

	var pulled string
	missing := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/tags":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"models":[]}`))
		case "/api/pull":
			defer r.Body.Close()
			var body map[string]any
			_ = json.NewDecoder(r.Body).Decode(&body)
			pulled, _ = body["name"].(string)
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"status":"success"}`))
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer missing.Close()

	pool := ollama.NewPool()
	available, unavailable, err := pool.EnsureModels(context.Background(), []ollama.ModelTarget{
		{BaseURL: present.URL, Name: "present:latest"},
		{BaseURL: missing.URL, Name: "missing:latest"},
	})
	if err != nil {
		t.Fatalf("EnsureModels() error = %v", err)
	}
	if pulled != "missing:latest" {
		t.Fatalf("pulled = %q", pulled)
	}
	if len(available) != 2 {
		t.Fatalf("available = %#v", available)
	}
	if len(unavailable) != 0 {
		t.Fatalf("unavailable = %#v", unavailable)
	}
}

func TestPoolContinuesWhenOneURLDown(t *testing.T) {
	up := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"models":[{"name":"ok:latest"}]}`))
	}))
	defer up.Close()

	pool := ollama.NewPool()
	available, unavailable, err := pool.EnsureModels(context.Background(), []ollama.ModelTarget{
		{BaseURL: up.URL, Name: "ok:latest"},
		{BaseURL: "http://127.0.0.1:1", Name: "down:latest"},
	})
	if err != nil {
		t.Fatalf("EnsureModels() error = %v", err)
	}
	if len(available) != 1 || available[0].Name != "ok:latest" {
		t.Fatalf("available = %#v", available)
	}
	if len(unavailable) != 1 || unavailable[0].Name != "down:latest" {
		t.Fatalf("unavailable = %#v", unavailable)
	}
}
