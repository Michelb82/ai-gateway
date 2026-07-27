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
	t.Setenv("OLLAMA_MODEL", "")
	t.Setenv("BRPOP_TIMEOUT", "")

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
	if cfg.OllamaModel != "qwen3:14b-q4_K_M" {
		t.Fatalf("OllamaModel = %q", cfg.OllamaModel)
	}
	if cfg.BRPopTimeout != 5 {
		t.Fatalf("BRPopTimeout = %d, want 5", cfg.BRPopTimeout)
	}
}

func TestLoadOverrides(t *testing.T) {
	t.Setenv("REDIS_ADDR", "localhost:6380")
	t.Setenv("INPUT_QUEUE", "custom.in")
	t.Setenv("OUTPUT_QUEUE", "custom.out")
	t.Setenv("OLLAMA_URL", "http://ollama:11434/")
	t.Setenv("OLLAMA_MODEL", "custom-model")
	t.Setenv("BRPOP_TIMEOUT", "10")

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
	if cfg.OllamaModel != "custom-model" {
		t.Fatalf("OllamaModel = %q", cfg.OllamaModel)
	}
	if cfg.BRPopTimeout != 10 {
		t.Fatalf("BRPopTimeout = %d", cfg.BRPopTimeout)
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
