package ollama_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/mywebsite/construction-ai-gateway/internal/ollama"
)

func TestCompleteSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat/completions" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Fatalf("method = %s", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(readFixture(t, "ollama_response.json"))
	}))
	defer server.Close()

	client := ollama.NewClient(server.URL)
	result, err := client.Complete(context.Background(), "system", "user prompt", "test-model", "")
	if err != nil {
		t.Fatalf("Complete() error = %v", err)
	}
	if result != "Electrical installations for homes" {
		t.Fatalf("result = %q", result)
	}
}

func TestCompleteHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusBadGateway)
	}))
	defer server.Close()

	client := ollama.NewClient(server.URL)
	_, err := client.Complete(context.Background(), "system", "user prompt", "test-model", "")
	if err == nil {
		t.Fatalf("Complete() expected error")
	}
}

func TestCompleteEmptyChoices(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[]}`))
	}))
	defer server.Close()

	client := ollama.NewClient(server.URL)
	_, err := client.Complete(context.Background(), "system", "user prompt", "test-model", "")
	if err == nil {
		t.Fatalf("Complete() expected error")
	}
}

func TestCompleteValidation(t *testing.T) {
	client := ollama.NewClient("http://example.com")

	_, err := client.Complete(context.Background(), "system", "", "model", "")
	if err == nil {
		t.Fatalf("Complete() expected error for empty prompt")
	}

	_, err = client.Complete(context.Background(), "", "prompt", "model", "")
	if err == nil {
		t.Fatalf("Complete() expected error for empty system prompt")
	}

	_, err = client.Complete(context.Background(), "system", "prompt", "", "")
	if err == nil {
		t.Fatalf("Complete() expected error for empty model")
	}
}

func TestCompleteKeepAlive(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if body["keep_alive"] != "2m" {
			t.Fatalf("keep_alive = %v", body["keep_alive"])
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(readFixture(t, "ollama_response.json"))
	}))
	defer server.Close()

	client := ollama.NewClient(server.URL)
	_, err := client.Complete(context.Background(), "system", "prompt", "model", "2m")
	if err != nil {
		t.Fatalf("Complete() error = %v", err)
	}
}

func TestModelAvailable(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/tags" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"models":[{"name":"qwen3:1.7b"},{"name":"qwen3:4b"}]}`))
	}))
	defer server.Close()

	client := ollama.NewClient(server.URL)
	ok, err := client.ModelAvailable(context.Background(), "qwen3:4b")
	if err != nil {
		t.Fatalf("ModelAvailable() error = %v", err)
	}
	if !ok {
		t.Fatalf("expected model available")
	}

	ok, err = client.ModelAvailable(context.Background(), "missing")
	if err != nil {
		t.Fatalf("ModelAvailable() error = %v", err)
	}
	if ok {
		t.Fatalf("expected model unavailable")
	}
}

func TestListModelsUnreachable(t *testing.T) {
	client := ollama.NewClient("http://127.0.0.1:1")
	_, err := client.ListModels(context.Background())
	if err == nil {
		t.Fatalf("ListModels() expected error")
	}
}

func TestEnsureModelsPullsMissing(t *testing.T) {
	var pulled []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/tags":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"models":[{"name":"present:latest"}]}`))
		case r.Method == http.MethodPost && r.URL.Path == "/api/pull":
			defer r.Body.Close()
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode pull body: %v", err)
			}
			if body["stream"] != false {
				t.Fatalf("stream = %v, want false", body["stream"])
			}
			name, _ := body["name"].(string)
			pulled = append(pulled, name)
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"status":"success"}`))
		default:
			t.Fatalf("unexpected %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	client := ollama.NewClient(server.URL)
	available, unavailable, err := client.EnsureModels(context.Background(), []string{"present:latest", "missing:latest", "present:latest"})
	if err != nil {
		t.Fatalf("EnsureModels() error = %v", err)
	}
	if len(pulled) != 1 || pulled[0] != "missing:latest" {
		t.Fatalf("pulled = %#v, want [missing:latest]", pulled)
	}
	if len(available) != 2 {
		t.Fatalf("available = %#v, want 2 models", available)
	}
	if len(unavailable) != 0 {
		t.Fatalf("unavailable = %#v, want none", unavailable)
	}
}

func TestEnsureModelsContinuesWhenOnePullFails(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/tags":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"models":[{"name":"present:latest"}]}`))
		case r.Method == http.MethodPost && r.URL.Path == "/api/pull":
			http.Error(w, "pull failed", http.StatusBadGateway)
		default:
			t.Fatalf("unexpected %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	client := ollama.NewClient(server.URL)
	available, unavailable, err := client.EnsureModels(context.Background(), []string{"present:latest", "missing:latest"})
	if err != nil {
		t.Fatalf("EnsureModels() error = %v", err)
	}
	if len(available) != 1 || available[0] != "present:latest" {
		t.Fatalf("available = %#v", available)
	}
	if len(unavailable) != 1 || unavailable[0] != "missing:latest" {
		t.Fatalf("unavailable = %#v", unavailable)
	}
}

func TestEnsureModelsFailsWhenOllamaDown(t *testing.T) {
	client := ollama.NewClient("http://127.0.0.1:1")
	_, _, err := client.EnsureModels(context.Background(), []string{"qwen3:1.7b"})
	if err == nil {
		t.Fatalf("EnsureModels() expected error")
	}
}

func TestEnsureModelsFailsWhenNoneAvailable(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/tags":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"models":[]}`))
		case r.Method == http.MethodPost && r.URL.Path == "/api/pull":
			http.Error(w, "pull failed", http.StatusBadGateway)
		default:
			t.Fatalf("unexpected %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	client := ollama.NewClient(server.URL)
	_, unavailable, err := client.EnsureModels(context.Background(), []string{"missing:latest"})
	if err == nil {
		t.Fatalf("EnsureModels() expected error")
	}
	if len(unavailable) != 1 {
		t.Fatalf("unavailable = %#v", unavailable)
	}
}

func TestPullHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "nope", http.StatusBadGateway)
	}))
	defer server.Close()

	client := ollama.NewClient(server.URL)
	if err := client.Pull(context.Background(), "missing"); err == nil {
		t.Fatalf("Pull() expected error")
	}
}

func readFixture(t *testing.T, name string) []byte {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("..", "..", "testdata", name))
	if err != nil {
		t.Fatalf("read fixture %s: %v", name, err)
	}
	return raw
}
