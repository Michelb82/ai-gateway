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
	t.Setenv("OLLAMA_MODEL_TRANSLATE", "")
	t.Setenv("OLLAMA_MODEL_ROUTING_TTL", "")
	t.Setenv("OLLAMA_MODEL_INTENT_TTL", "")
	t.Setenv("OLLAMA_MODEL_TRANSLATE_TTL", "")
	t.Setenv("HTTP_ADDR", "")
	t.Setenv("BRPOP_TIMEOUT", "")
	t.Setenv("DEBUG", "")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.OllamaModelRouting != "qwen3:1.7b-q4_K_M" {
		t.Fatalf("OllamaModelRouting = %q", cfg.OllamaModelRouting)
	}
	if cfg.OllamaModelRoutingTTL != "5m" {
		t.Fatalf("OllamaModelRoutingTTL = %q", cfg.OllamaModelRoutingTTL)
	}
	if cfg.OllamaModelIntentTTL != "5m" {
		t.Fatalf("OllamaModelIntentTTL = %q", cfg.OllamaModelIntentTTL)
	}
	if cfg.OllamaModelTranslateTTL != "2m" {
		t.Fatalf("OllamaModelTranslateTTL = %q", cfg.OllamaModelTranslateTTL)
	}
}

func TestLoadOverrides(t *testing.T) {
	t.Setenv("REDIS_ADDR", "localhost:6380")
	t.Setenv("INPUT_QUEUE", "custom.in")
	t.Setenv("OUTPUT_QUEUE", "custom.out")
	t.Setenv("OLLAMA_URL", "http://ollama:11434/")
	t.Setenv("OLLAMA_MODEL_ROUTING", "routing-model")
	t.Setenv("OLLAMA_MODEL_INTENT", "intent-model")
	t.Setenv("OLLAMA_MODEL_TRANSLATE", "translate-model")
	t.Setenv("OLLAMA_MODEL_ROUTING_TTL", "10m")
	t.Setenv("OLLAMA_MODEL_INTENT_TTL", "3m")
	t.Setenv("OLLAMA_MODEL_TRANSLATE_TTL", "90s")
	t.Setenv("HTTP_ADDR", ":8080")
	t.Setenv("BRPOP_TIMEOUT", "10")
	t.Setenv("DEBUG", "true")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.OllamaModelTranslate != "translate-model" {
		t.Fatalf("OllamaModelTranslate = %q", cfg.OllamaModelTranslate)
	}
	if cfg.OllamaModelRoutingTTL != "10m" {
		t.Fatalf("OllamaModelRoutingTTL = %q", cfg.OllamaModelRoutingTTL)
	}
	if cfg.OllamaModelIntentTTL != "3m" {
		t.Fatalf("OllamaModelIntentTTL = %q", cfg.OllamaModelIntentTTL)
	}
	if cfg.OllamaModelTranslateTTL != "90s" {
		t.Fatalf("OllamaModelTranslateTTL = %q", cfg.OllamaModelTranslateTTL)
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
