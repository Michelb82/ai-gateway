package config_test

import (
	"os"
	"testing"

	"github.com/buildright/construction-ai-gateway/internal/config"
)

func TestLoadDefaults(t *testing.T) {
	t.Setenv("REDIS_ADDR", "")
	t.Setenv("INPUT_QUEUE", "")
	t.Setenv("OUTPUT_QUEUE", "")
	t.Setenv("OLLAMA_URL", "")
	t.Setenv("OLLAMA_MODEL_ROUTING", "")
	t.Setenv("OLLAMA_MODEL_INTENT", "")
	t.Setenv("HTTP_ADDR", "")
	t.Setenv("BRPOP_TIMEOUT", "")
	t.Setenv("DEBUG", "")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.RedisAddr != "redis:6379" {
		t.Fatalf("RedisAddr = %q, want redis:6379", cfg.RedisAddr)
	}
	if cfg.InputQueue != "ai.requests" {
		t.Fatalf("InputQueue = %q, want ai.requests", cfg.InputQueue)
	}
	if cfg.OutputQueue != "ai.responses" {
		t.Fatalf("OutputQueue = %q, want ai.responses", cfg.OutputQueue)
	}
	if cfg.OllamaURL != "http://foundation-model:11434" {
		t.Fatalf("OllamaURL = %q", cfg.OllamaURL)
	}
	if cfg.OllamaModelRouting != "qwen3:1.7b" {
		t.Fatalf("OllamaModelRouting = %q", cfg.OllamaModelRouting)
	}
	if cfg.OllamaModelIntent != "qwen3:4b" {
		t.Fatalf("OllamaModelIntent = %q", cfg.OllamaModelIntent)
	}
	if cfg.HTTPAddr != ":80" {
		t.Fatalf("HTTPAddr = %q", cfg.HTTPAddr)
	}
	if cfg.BRPopTimeout != 5 {
		t.Fatalf("BRPopTimeout = %d, want 5", cfg.BRPopTimeout)
	}
	if cfg.Debug {
		t.Fatalf("Debug = true, want false")
	}
}

func TestLoadOverrides(t *testing.T) {
	t.Setenv("REDIS_ADDR", "localhost:6380")
	t.Setenv("INPUT_QUEUE", "custom.in")
	t.Setenv("OUTPUT_QUEUE", "custom.out")
	t.Setenv("OLLAMA_URL", "http://ollama:11434/")
	t.Setenv("OLLAMA_MODEL_ROUTING", "routing-model")
	t.Setenv("OLLAMA_MODEL_INTENT", "intent-model")
	t.Setenv("HTTP_ADDR", ":8080")
	t.Setenv("BRPOP_TIMEOUT", "10")
	t.Setenv("DEBUG", "true")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.RedisAddr != "localhost:6380" {
		t.Fatalf("RedisAddr = %q", cfg.RedisAddr)
	}
	if cfg.InputQueue != "custom.in" {
		t.Fatalf("InputQueue = %q", cfg.InputQueue)
	}
	if cfg.OutputQueue != "custom.out" {
		t.Fatalf("OutputQueue = %q", cfg.OutputQueue)
	}
	if cfg.OllamaURL != "http://ollama:11434" {
		t.Fatalf("OllamaURL = %q", cfg.OllamaURL)
	}
	if cfg.OllamaModelRouting != "routing-model" {
		t.Fatalf("OllamaModelRouting = %q", cfg.OllamaModelRouting)
	}
	if cfg.OllamaModelIntent != "intent-model" {
		t.Fatalf("OllamaModelIntent = %q", cfg.OllamaModelIntent)
	}
	if cfg.HTTPAddr != ":8080" {
		t.Fatalf("HTTPAddr = %q", cfg.HTTPAddr)
	}
	if cfg.BRPopTimeout != 10 {
		t.Fatalf("BRPopTimeout = %d", cfg.BRPopTimeout)
	}
	if !cfg.Debug {
		t.Fatalf("Debug = false, want true")
	}
}

func TestLoadInvalidBRPopTimeout(t *testing.T) {
	tests := []struct {
		name  string
		value string
	}{
		{name: "non numeric", value: "abc"},
		{name: "zero", value: "0"},
		{name: "negative", value: "-1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("BRPOP_TIMEOUT", tt.value)
			_, err := config.Load()
			if err == nil {
				t.Fatalf("Load() expected error for BRPOP_TIMEOUT=%q", tt.value)
			}
		})
	}
}

func TestMain(m *testing.M) {
	os.Exit(m.Run())
}
