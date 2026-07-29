package config_test

import (
	"os"
	"testing"

	"github.com/mywebsite/construction-ai-gateway/internal/config"
)

func TestLoadDefaults(t *testing.T) {
	clearLLMEnv(t)

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.LLMURLRouting != "http://llm-model:11434" {
		t.Fatalf("LLMURLRouting = %q", cfg.LLMURLRouting)
	}
	if cfg.LLMURLIntent != "http://llm-model:11434" {
		t.Fatalf("LLMURLIntent = %q", cfg.LLMURLIntent)
	}
	if cfg.LLMURLTranslate != "http://llm-model:11434" {
		t.Fatalf("LLMURLTranslate = %q", cfg.LLMURLTranslate)
	}
	if cfg.CloudEventTypePrefix != "com.mywebsite.ai" {
		t.Fatalf("CloudEventTypePrefix = %q", cfg.CloudEventTypePrefix)
	}
	if cfg.LLMModelRouting != "qwen3:1.7b-q4_K_M" {
		t.Fatalf("LLMModelRouting = %q", cfg.LLMModelRouting)
	}
	if cfg.LLMModelRoutingTTL != "5m" {
		t.Fatalf("LLMModelRoutingTTL = %q", cfg.LLMModelRoutingTTL)
	}
	if cfg.LLMModelIntentTTL != "5m" {
		t.Fatalf("LLMModelIntentTTL = %q", cfg.LLMModelIntentTTL)
	}
	if cfg.LLMModelTranslateTTL != "2m" {
		t.Fatalf("LLMModelTranslateTTL = %q", cfg.LLMModelTranslateTTL)
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
	t.Setenv("LLM_URL_ROUTING", "http://routing:11434/")
	t.Setenv("LLM_URL_INTENT", "http://intent:11434/")
	t.Setenv("LLM_URL_TRANSLATE", "http://translate:11434/")
	t.Setenv("LLM_MODEL_ROUTING", "routing-model")
	t.Setenv("LLM_MODEL_INTENT", "intent-model")
	t.Setenv("LLM_MODEL_TRANSLATE", "translate-model")
	t.Setenv("LLM_MODEL_ROUTING_TTL", "10m")
	t.Setenv("LLM_MODEL_INTENT_TTL", "3m")
	t.Setenv("LLM_MODEL_TRANSLATE_TTL", "90s")
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

	if cfg.LLMURLRouting != "http://routing:11434" {
		t.Fatalf("LLMURLRouting = %q", cfg.LLMURLRouting)
	}
	if cfg.LLMURLIntent != "http://intent:11434" {
		t.Fatalf("LLMURLIntent = %q", cfg.LLMURLIntent)
	}
	if cfg.LLMModelTranslate != "translate-model" {
		t.Fatalf("LLMModelTranslate = %q", cfg.LLMModelTranslate)
	}
	if cfg.CloudEventTypePrefix != "com.example.ai" {
		t.Fatalf("CloudEventTypePrefix = %q", cfg.CloudEventTypePrefix)
	}
	if cfg.LLMModelRoutingTTL != "10m" {
		t.Fatalf("LLMModelRoutingTTL = %q", cfg.LLMModelRoutingTTL)
	}
	if cfg.PriorityHighCount != 5 {
		t.Fatalf("PriorityHighCount = %d, want 5", cfg.PriorityHighCount)
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

func clearLLMEnv(t *testing.T) {
	t.Helper()
	keys := []string{
		"REDIS_ADDR", "INPUT_QUEUE", "OUTPUT_QUEUE",
		"LLM_URL_ROUTING", "LLM_URL_INTENT", "LLM_URL_TRANSLATE",
		"LLM_MODEL_ROUTING", "LLM_MODEL_INTENT", "LLM_MODEL_TRANSLATE",
		"LLM_MODEL_ROUTING_TTL", "LLM_MODEL_INTENT_TTL", "LLM_MODEL_TRANSLATE_TTL",
		"CLOUDEVENT_TYPE_PREFIX", "HTTP_ADDR", "BRPOP_TIMEOUT",
		"PRIORITY_HIGH_COUNT", "PRIORITY_MEDIUM_COUNT", "DEBUG",
	}
	for _, key := range keys {
		t.Setenv(key, "")
	}
}

func TestMain(m *testing.M) {
	os.Exit(m.Run())
}
