package config_test

import (
	"os"
	"testing"

	"github.com/mywebsite/construction-ai-gateway/internal/config"
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
	t.Setenv("CLOUDEVENT_TYPE_PREFIX", "")
	t.Setenv("HTTP_ADDR", "")
	t.Setenv("BRPOP_TIMEOUT", "")
	t.Setenv("PRIORITY_HIGH_COUNT", "")
	t.Setenv("PRIORITY_MEDIUM_COUNT", "")
	t.Setenv("DEBUG", "")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.OllamaURL != "http://llm-model:11434" {
		t.Fatalf("OllamaURL = %q", cfg.OllamaURL)
	}
	if cfg.CloudEventTypePrefix != "com.mywebsite.ai" {
		t.Fatalf("CloudEventTypePrefix = %q", cfg.CloudEventTypePrefix)
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
	if cfg.PriorityHighCount != 3 {
		t.Fatalf("PriorityHighCount = %d, want 3", cfg.PriorityHighCount)
	}
	if cfg.PriorityMediumCount != 3 {
		t.Fatalf("PriorityMediumCount = %d, want 3", cfg.PriorityMediumCount)
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
	t.Setenv("CLOUDEVENT_TYPE_PREFIX", "com.example.ai")
	t.Setenv("HTTP_ADDR", ":8080")
	t.Setenv("BRPOP_TIMEOUT", "10")
	t.Setenv("PRIORITY_HIGH_COUNT", "5")
	t.Setenv("PRIORITY_MEDIUM_COUNT", "4")
	t.Setenv("DEBUG", "true")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.OllamaModelTranslate != "translate-model" {
		t.Fatalf("OllamaModelTranslate = %q", cfg.OllamaModelTranslate)
	}
	if cfg.CloudEventTypePrefix != "com.example.ai" {
		t.Fatalf("CloudEventTypePrefix = %q", cfg.CloudEventTypePrefix)
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
	if cfg.PriorityHighCount != 5 {
		t.Fatalf("PriorityHighCount = %d, want 5", cfg.PriorityHighCount)
	}
	if cfg.PriorityMediumCount != 4 {
		t.Fatalf("PriorityMediumCount = %d, want 4", cfg.PriorityMediumCount)
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

func TestLoadInvalidPriorityCounts(t *testing.T) {
	tests := []struct {
		name string
		key  string
		val  string
	}{
		{name: "high non numeric", key: "PRIORITY_HIGH_COUNT", val: "abc"},
		{name: "high zero", key: "PRIORITY_HIGH_COUNT", val: "0"},
		{name: "medium negative", key: "PRIORITY_MEDIUM_COUNT", val: "-1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("PRIORITY_HIGH_COUNT", "")
			t.Setenv("PRIORITY_MEDIUM_COUNT", "")
			t.Setenv(tt.key, tt.val)
			_, err := config.Load()
			if err == nil {
				t.Fatalf("Load() expected error for %s=%q", tt.key, tt.val)
			}
		})
	}
}

func TestMain(m *testing.M) {
	os.Exit(m.Run())
}
