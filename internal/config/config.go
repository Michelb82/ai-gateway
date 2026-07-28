package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	RedisAddr    string
	InputQueue   string
	OutputQueue  string
	OllamaURL    string
	OllamaModel  string
	BRPopTimeout int
	Debug        bool
}

func Load() (Config, error) {
	cfg := Config{
		RedisAddr:    envOrDefault("REDIS_ADDR", "redis:6379"),
		InputQueue:   envOrDefault("INPUT_QUEUE", "ai.requests"),
		OutputQueue:  envOrDefault("OUTPUT_QUEUE", "ai.responses"),
		OllamaURL:    strings.TrimRight(envOrDefault("OLLAMA_URL", "http://foundation-model:11434"), "/"),
		OllamaModel:  envOrDefault("OLLAMA_MODEL", "qwen3:14b-q4_K_M"),
		BRPopTimeout: 5,
		Debug:        envBool("DEBUG"),
	}

	timeoutRaw := strings.TrimSpace(os.Getenv("BRPOP_TIMEOUT"))
	if timeoutRaw != "" {
		timeout, err := strconv.Atoi(timeoutRaw)
		if err != nil || timeout < 1 {
			return Config{}, fmt.Errorf("invalid BRPOP_TIMEOUT %q: must be a positive integer", timeoutRaw)
		}
		cfg.BRPopTimeout = timeout
	}

	if cfg.RedisAddr == "" {
		return Config{}, fmt.Errorf("REDIS_ADDR must not be blank")
	}
	if cfg.InputQueue == "" {
		return Config{}, fmt.Errorf("INPUT_QUEUE must not be blank")
	}
	if cfg.OutputQueue == "" {
		return Config{}, fmt.Errorf("OUTPUT_QUEUE must not be blank")
	}
	if cfg.OllamaURL == "" {
		return Config{}, fmt.Errorf("OLLAMA_URL must not be blank")
	}
	if cfg.OllamaModel == "" {
		return Config{}, fmt.Errorf("OLLAMA_MODEL must not be blank")
	}

	return cfg, nil
}

func envOrDefault(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

func envBool(key string) bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv(key))) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}
