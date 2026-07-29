package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	RedisAddr               string
	InputQueue              string
	OutputQueue             string
	OllamaURL               string
	OllamaModelRouting      string
	OllamaModelIntent       string
	OllamaModelTranslate    string
	OllamaModelRoutingTTL   string
	OllamaModelIntentTTL    string
	OllamaModelTranslateTTL string
	CloudEventTypePrefix    string
	HTTPAddr                string
	BRPopTimeout            int
	PriorityHighCount       int
	PriorityMediumCount     int
	Debug                   bool
}

func Load() (Config, error) {
	cfg := Config{
		RedisAddr:               envOrDefault("REDIS_ADDR", "redis:6379"),
		InputQueue:              envOrDefault("INPUT_QUEUE", "ai.requests"),
		OutputQueue:             envOrDefault("OUTPUT_QUEUE", "ai.responses"),
		OllamaURL:               strings.TrimRight(envOrDefault("OLLAMA_URL", "http://llm-model:11434"), "/"),
		OllamaModelRouting:      envOrDefault("OLLAMA_MODEL_ROUTING", "qwen3:1.7b-q4_K_M"),
		OllamaModelIntent:       envOrDefault("OLLAMA_MODEL_INTENT", "qwen3:4b-q4_K_M"),
		OllamaModelTranslate:    envOrDefault("OLLAMA_MODEL_TRANSLATE", "qwen3:14b-q4_K_M"),
		OllamaModelRoutingTTL:   envOrDefault("OLLAMA_MODEL_ROUTING_TTL", "5m"),
		OllamaModelIntentTTL:    envOrDefault("OLLAMA_MODEL_INTENT_TTL", "5m"),
		OllamaModelTranslateTTL: envOrDefault("OLLAMA_MODEL_TRANSLATE_TTL", "2m"),
		CloudEventTypePrefix:    envOrDefault("CLOUDEVENT_TYPE_PREFIX", "com.mywebsite.ai"),
		HTTPAddr:                envOrDefault("HTTP_ADDR", ":80"),
		BRPopTimeout:            5,
		PriorityHighCount:       3,
		PriorityMediumCount:     3,
		Debug:                   envBool("DEBUG"),
	}

	timeoutRaw := strings.TrimSpace(os.Getenv("BRPOP_TIMEOUT"))
	if timeoutRaw != "" {
		timeout, err := strconv.Atoi(timeoutRaw)
		if err != nil || timeout < 1 {
			return Config{}, fmt.Errorf("invalid BRPOP_TIMEOUT %q: must be a positive integer", timeoutRaw)
		}
		cfg.BRPopTimeout = timeout
	}

	highCount, err := envPositiveInt("PRIORITY_HIGH_COUNT", cfg.PriorityHighCount)
	if err != nil {
		return Config{}, err
	}
	cfg.PriorityHighCount = highCount

	mediumCount, err := envPositiveInt("PRIORITY_MEDIUM_COUNT", cfg.PriorityMediumCount)
	if err != nil {
		return Config{}, err
	}
	cfg.PriorityMediumCount = mediumCount

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
	if cfg.OllamaModelRouting == "" {
		return Config{}, fmt.Errorf("OLLAMA_MODEL_ROUTING must not be blank")
	}
	if cfg.OllamaModelIntent == "" {
		return Config{}, fmt.Errorf("OLLAMA_MODEL_INTENT must not be blank")
	}
	if cfg.OllamaModelTranslate == "" {
		return Config{}, fmt.Errorf("OLLAMA_MODEL_TRANSLATE must not be blank")
	}
	if cfg.OllamaModelRoutingTTL == "" {
		return Config{}, fmt.Errorf("OLLAMA_MODEL_ROUTING_TTL must not be blank")
	}
	if cfg.OllamaModelIntentTTL == "" {
		return Config{}, fmt.Errorf("OLLAMA_MODEL_INTENT_TTL must not be blank")
	}
	if cfg.OllamaModelTranslateTTL == "" {
		return Config{}, fmt.Errorf("OLLAMA_MODEL_TRANSLATE_TTL must not be blank")
	}
	if cfg.CloudEventTypePrefix == "" {
		return Config{}, fmt.Errorf("CLOUDEVENT_TYPE_PREFIX must not be blank")
	}
	if cfg.HTTPAddr == "" {
		return Config{}, fmt.Errorf("HTTP_ADDR must not be blank")
	}

	return cfg, nil
}

func envOrDefault(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

func envPositiveInt(key string, fallback int) (int, error) {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback, nil
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value < 1 {
		return 0, fmt.Errorf("invalid %s %q: must be a positive integer", key, raw)
	}
	return value, nil
}

func envBool(key string) bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv(key))) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}
