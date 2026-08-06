package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	RedisAddr            string
	InputQueue           string
	OutputQueue          string
	LLMURLRouting        string
	LLMURLIntent         string
	LLMURLTranslate      string
	LLMModelRouting      string
	LLMModelIntent       string
	LLMModelTranslate    string
	LLMModelRoutingTTL   string
	LLMModelIntentTTL    string
	LLMModelTranslateTTL string
	LLMMaxCharsRouting   int
	LLMMaxCharsIntent    int
	LLMMaxCharsTranslate int
	CloudEventTypePrefix string
	HTTPAddr             string
	BRPopTimeout         int
	PriorityHighCount    int
	PriorityMediumCount  int
	Debug                bool
}

func Load() (Config, error) {
	cfg := Config{
		RedisAddr:            envOrDefault("REDIS_ADDR", "redis:6379"),
		InputQueue:           envOrDefault("INPUT_QUEUE", "ai.requests"),
		OutputQueue:          envOrDefault("OUTPUT_QUEUE", "ai.responses"),
		LLMURLRouting:        strings.TrimRight(envOrDefault("LLM_URL_ROUTING", "http://llm-model:11434"), "/"),
		LLMURLIntent:         strings.TrimRight(envOrDefault("LLM_URL_INTENT", "http://llm-model:11434"), "/"),
		LLMURLTranslate:      strings.TrimRight(envOrDefault("LLM_URL_TRANSLATE", "http://llm-model:11434"), "/"),
		LLMModelRouting:      envOrDefault("LLM_MODEL_ROUTING", "qwen3:1.7b-q4_K_M"),
		LLMModelIntent:       envOrDefault("LLM_MODEL_INTENT", "qwen3:4b-q4_K_M"),
		LLMModelTranslate:    envOrDefault("LLM_MODEL_TRANSLATE", "qwen3:14b-q4_K_M"),
		LLMModelRoutingTTL:   envOrDefault("LLM_MODEL_ROUTING_TTL", "5m"),
		LLMModelIntentTTL:    envOrDefault("LLM_MODEL_INTENT_TTL", "5m"),
		LLMModelTranslateTTL: envOrDefault("LLM_MODEL_TRANSLATE_TTL", "2m"),
		LLMMaxCharsRouting:   200,
		LLMMaxCharsIntent:    8000,
		LLMMaxCharsTranslate: 16000,
		CloudEventTypePrefix: envOrDefault("CLOUDEVENT_TYPE_PREFIX", "com.mywebsite.ai"),
		HTTPAddr:             envOrDefault("HTTP_ADDR", ":80"),
		BRPopTimeout:         5,
		PriorityHighCount:    3,
		PriorityMediumCount:  3,
		Debug:                envBool("DEBUG"),
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

	maxCharsRouting, err := envPositiveInt("LLM_MAX_CHARS_ROUTING", cfg.LLMMaxCharsRouting)
	if err != nil {
		return Config{}, err
	}
	cfg.LLMMaxCharsRouting = maxCharsRouting

	maxCharsIntent, err := envPositiveInt("LLM_MAX_CHARS_INTENT", cfg.LLMMaxCharsIntent)
	if err != nil {
		return Config{}, err
	}
	cfg.LLMMaxCharsIntent = maxCharsIntent

	maxCharsTranslate, err := envPositiveInt("LLM_MAX_CHARS_TRANSLATE", cfg.LLMMaxCharsTranslate)
	if err != nil {
		return Config{}, err
	}
	cfg.LLMMaxCharsTranslate = maxCharsTranslate

	required := []struct {
		name  string
		value string
	}{
		{"REDIS_ADDR", cfg.RedisAddr},
		{"INPUT_QUEUE", cfg.InputQueue},
		{"OUTPUT_QUEUE", cfg.OutputQueue},
		{"LLM_URL_ROUTING", cfg.LLMURLRouting},
		{"LLM_URL_INTENT", cfg.LLMURLIntent},
		{"LLM_URL_TRANSLATE", cfg.LLMURLTranslate},
		{"LLM_MODEL_ROUTING", cfg.LLMModelRouting},
		{"LLM_MODEL_INTENT", cfg.LLMModelIntent},
		{"LLM_MODEL_TRANSLATE", cfg.LLMModelTranslate},
		{"LLM_MODEL_ROUTING_TTL", cfg.LLMModelRoutingTTL},
		{"LLM_MODEL_INTENT_TTL", cfg.LLMModelIntentTTL},
		{"LLM_MODEL_TRANSLATE_TTL", cfg.LLMModelTranslateTTL},
		{"CLOUDEVENT_TYPE_PREFIX", cfg.CloudEventTypePrefix},
		{"HTTP_ADDR", cfg.HTTPAddr},
	}
	for _, item := range required {
		if item.value == "" {
			return Config{}, fmt.Errorf("%s must not be blank", item.name)
		}
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
