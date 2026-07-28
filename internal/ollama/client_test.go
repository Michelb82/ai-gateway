package ollama_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/buildright/construction-ai-gateway/internal/ollama"
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
	result, err := client.Complete(context.Background(), "system", "user prompt", "test-model")
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
	_, err := client.Complete(context.Background(), "system", "user prompt", "test-model")
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
	_, err := client.Complete(context.Background(), "system", "user prompt", "test-model")
	if err == nil {
		t.Fatalf("Complete() expected error")
	}
}

func TestCompleteValidation(t *testing.T) {
	client := ollama.NewClient("http://example.com")

	_, err := client.Complete(context.Background(), "system", "", "model")
	if err == nil {
		t.Fatalf("Complete() expected error for empty prompt")
	}

	_, err = client.Complete(context.Background(), "", "prompt", "model")
	if err == nil {
		t.Fatalf("Complete() expected error for empty system prompt")
	}

	_, err = client.Complete(context.Background(), "system", "prompt", "")
	if err == nil {
		t.Fatalf("Complete() expected error for empty model")
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

func readFixture(t *testing.T, name string) []byte {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("..", "..", "testdata", name))
	if err != nil {
		t.Fatalf("read fixture %s: %v", name, err)
	}
	return raw
}
